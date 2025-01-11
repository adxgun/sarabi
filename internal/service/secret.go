package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"sarabi/internal/database"
	"sarabi/internal/misc"
	"sarabi/internal/types"
	"sarabi/logger"
	"time"
)

type SecretService interface {
	Create(ctx context.Context, params types.CreateSecretParams) (*types.Secret, error)
	CreateAll(ctx context.Context, params ...types.CreateSecretParams) ([]*types.Secret, error)
	FindAll(ctx context.Context, applicationID uuid.UUID) ([]*types.Secret, error)
	CreateDeploymentSecrets(ctx context.Context, deploymentID uuid.UUID, secrets []*types.Secret) error
	FindDeploymentSecrets(ctx context.Context, deploymentID uuid.UUID) ([]*types.Secret, error)
	CreateServerConfig(ctx context.Context, params types.CreateServerConfigParams) (*types.ServerConfigResponse, error)
	FindApplicationServerConfigs(ctx context.Context, applicationID uuid.UUID) ([]*types.ServerConfig, error)
	DeleteDeploymentSecrets(ctx context.Context, deploymentID uuid.UUID) error
	FindStorageCredentials(ctx context.Context, applicationID uuid.UUID) (*types.StorageCredentials, error)
}

type secretService struct {
	encryptor                  misc.Encryptor
	repository                 database.SecretRepository
	deploymentSecretRepository database.DeploymentSecretRepository
	serverConfigRepository     database.ServerConfigRepository
}

func NewSecretService(enc misc.Encryptor, repo database.SecretRepository,
	depSecretRepo database.DeploymentSecretRepository, serverConfigRepository database.ServerConfigRepository) SecretService {
	return &secretService{
		encryptor:                  enc,
		repository:                 repo,
		deploymentSecretRepository: depSecretRepo,
		serverConfigRepository:     serverConfigRepository,
	}
}

func (s *secretService) Create(ctx context.Context, params types.CreateSecretParams) (*types.Secret, error) {
	encryptedValue, err := s.encryptor.Encrypt(params.Value)
	if err != nil {
		return nil, err
	}

	if sc, err := s.repository.FindBy(ctx, params.ApplicationID, params.Key, params.Environment, string(params.InstanceType)); err == nil && sc.ID != uuid.Nil {
		err = s.repository.UpdateValue(ctx, sc.ID, encryptedValue)
		if err != nil {
			return nil, err
		}
		sc.Value = params.Value
		return sc, nil
	}

	sc := &types.Secret{
		ID:            uuid.New(),
		ApplicationID: params.ApplicationID,
		Name:          params.Key,
		Value:         encryptedValue,
		Environment:   params.Environment,
		InstanceType:  string(params.InstanceType),
	}
	if err := s.repository.Save(ctx, sc); err != nil {
		return nil, err
	}

	sc.Value = params.Value
	return sc, nil
}

func (s *secretService) CreateAll(ctx context.Context, values ...types.CreateSecretParams) ([]*types.Secret, error) {
	result := make([]*types.Secret, 0, len(values))
	for _, param := range values {
		r, err := s.Create(ctx, param)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, nil
}

func (s *secretService) FindAll(ctx context.Context, applicationID uuid.UUID) ([]*types.Secret, error) {
	secrets, err := s.repository.FindAll(ctx, applicationID)
	if err != nil {
		return nil, err
	}

	for _, ss := range secrets {
		decrypted, err := s.encryptor.Decrypt(ss.Value)
		if err != nil {
			return nil, err
		}
		ss.Value = decrypted
	}
	return secrets, nil
}

func (s *secretService) CreateDeploymentSecrets(ctx context.Context, deploymentID uuid.UUID, secrets []*types.Secret) error {
	values := make([]*types.DeploymentSecret, 0, len(secrets))
	for _, ss := range secrets {
		values = append(values, &types.DeploymentSecret{
			ID:           uuid.New(),
			DeploymentID: deploymentID,
			SecretID:     ss.ID,
		})
	}
	return s.deploymentSecretRepository.SaveAll(ctx, values)
}

func (s *secretService) FindDeploymentSecrets(ctx context.Context, deploymentID uuid.UUID) ([]*types.Secret, error) {
	values, err := s.deploymentSecretRepository.FindAll(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	secrets := make([]*types.Secret, 0, len(values))
	for _, v := range values {
		decrypted, err := s.encryptor.Decrypt(v.Secret.Value)
		if err != nil {
			return nil, err
		}

		secrets = append(secrets, &types.Secret{
			ID:            v.ID,
			ApplicationID: v.Secret.ApplicationID,
			Name:          v.Secret.Name,
			Value:         decrypted,
			Environment:   v.Secret.Environment,
			InstanceType:  v.Secret.InstanceType,
		})
	}
	return secrets, nil
}

func (s *secretService) CreateServerConfig(ctx context.Context, params types.CreateServerConfigParams) (*types.ServerConfigResponse, error) {
	logger.Info("received request to create server config",
		zap.Any("application_id", params.ApplicationID),
		zap.String("provider", params.Provider),
		zap.String("name", params.Name))

	serializedValue, err := json.Marshal(params.Value)
	if err != nil {
		return nil, err
	}

	encrypted, err := s.encryptor.Encrypt(string(serializedValue))
	if err != nil {
		return nil, err
	}

	existing, err := s.serverConfigRepository.FindByName(ctx, params.ApplicationID, params.Provider, params.Name)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		sConfig := &types.ServerConfig{
			ID:            uuid.New(),
			ApplicationID: params.ApplicationID,
			Provider:      params.Provider,
			Name:          params.Name,
			Value:         encrypted,
			CreatedAt:     time.Now(),
		}
		err = s.serverConfigRepository.Save(ctx, sConfig)
		if err != nil {
			return nil, err
		}
		return &types.ServerConfigResponse{ID: sConfig.ID}, nil
	} else if err == nil {
		err = s.serverConfigRepository.UpdateServerConfigValue(ctx, existing.ID, encrypted)
		if err != nil {
			return nil, err
		}
		return &types.ServerConfigResponse{ID: existing.ID}, nil
	} else {
		return nil, err
	}
}

func (s *secretService) FindApplicationServerConfigs(ctx context.Context, applicationID uuid.UUID) ([]*types.ServerConfig, error) {
	all, err := s.serverConfigRepository.FindByApplicationID(ctx, applicationID)
	if err != nil {
		return nil, err
	}

	result := make([]*types.ServerConfig, 0, len(all))
	for _, next := range all {
		decryptedValue, err := s.encryptor.Decrypt(next.Value)
		if err != nil {
			return nil, err
		}
		next.Value = decryptedValue
		result = append(result, next)
	}
	return result, nil
}

func (s *secretService) DeleteDeploymentSecrets(ctx context.Context, deploymentID uuid.UUID) error {
	vars, err := s.deploymentSecretRepository.FindAll(ctx, deploymentID)
	if err != nil {
		return err
	}

	for _, v := range vars {
		if err := s.deploymentSecretRepository.Delete(ctx, v.ID); err != nil {
			return err
		}
	}
	return err
}

func (s *secretService) FindStorageCredentials(ctx context.Context, applicationID uuid.UUID) (*types.StorageCredentials, error) {
	credentials, err := s.FindApplicationServerConfigs(ctx, applicationID)
	if err != nil {
		return nil, err
	}

	objectStorageConfig := lo.Filter(credentials, func(item *types.ServerConfig, index int) bool {
		return item.Name == types.ServerConfigObjectStorage
	})
	if len(objectStorageConfig) == 0 {
		return nil, errors.New("no object storage configured")
	}

	value := objectStorageConfig[0]
	cred := &types.StorageCredentials{}
	if err := json.Unmarshal([]byte(value.Value), cred); err != nil {
		return nil, err
	}
	return cred, nil
}

func FindSecret(name string, secrets []*types.Secret) (*types.Secret, error) {
	for _, next := range secrets {
		if next.Name == name {
			return next, nil
		}
	}
	return nil, fmt.Errorf("secret: %s was not found", name)
}
