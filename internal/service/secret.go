package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"sarabi"
	"sarabi/internal/database"
	types2 "sarabi/internal/types"
	"time"
)

type SecretService interface {
	Create(ctx context.Context, params types2.CreateSecretParams) (*types2.Secret, error)
	CreateAll(ctx context.Context, params ...types2.CreateSecretParams) ([]*types2.Secret, error)
	FindAll(ctx context.Context, applicationID uuid.UUID) ([]*types2.Secret, error)
	CreateDeploymentSecrets(ctx context.Context, deploymentID uuid.UUID, secrets []*types2.Secret) error
	FindDeploymentSecrets(ctx context.Context, deploymentID uuid.UUID) ([]*types2.Secret, error)
	CreateCredentials(ctx context.Context, params types2.AddCredentialsParams) ([]*types2.Credential, error)
	FindApplicationCredentials(ctx context.Context, applicationID uuid.UUID, provider string) ([]*types2.Credential, error)
	DeleteDeploymentSecrets(ctx context.Context, deploymentID uuid.UUID) error
}

type secretService struct {
	encryptor                  sarabi.Encryptor
	repository                 database.SecretRepository
	deploymentSecretRepository database.DeploymentSecretRepository
	credentialRepository       database.CredentialRepository
}

func NewSecretService(enc sarabi.Encryptor, repo database.SecretRepository,
	depSecretRepo database.DeploymentSecretRepository, credentialRepo database.CredentialRepository) SecretService {
	return &secretService{
		encryptor:                  enc,
		repository:                 repo,
		deploymentSecretRepository: depSecretRepo,
		credentialRepository:       credentialRepo,
	}
}

func (s *secretService) Create(ctx context.Context, params types2.CreateSecretParams) (*types2.Secret, error) {
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

	sc := &types2.Secret{
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

func (s *secretService) CreateAll(ctx context.Context, values ...types2.CreateSecretParams) ([]*types2.Secret, error) {
	result := make([]*types2.Secret, 0, len(values))
	for _, param := range values {
		r, err := s.Create(ctx, param)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, nil
}

func (s *secretService) FindAll(ctx context.Context, applicationID uuid.UUID) ([]*types2.Secret, error) {
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

func (s *secretService) CreateDeploymentSecrets(ctx context.Context, deploymentID uuid.UUID, secrets []*types2.Secret) error {
	values := make([]*types2.DeploymentSecret, 0, len(secrets))
	for _, ss := range secrets {
		values = append(values, &types2.DeploymentSecret{
			ID:           uuid.New(),
			DeploymentID: deploymentID,
			SecretID:     ss.ID,
		})
	}
	return s.deploymentSecretRepository.SaveAll(ctx, values)
}

func (s *secretService) FindDeploymentSecrets(ctx context.Context, deploymentID uuid.UUID) ([]*types2.Secret, error) {
	values, err := s.deploymentSecretRepository.FindAll(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	secrets := make([]*types2.Secret, 0, len(values))
	for _, v := range values {
		decrypted, err := s.encryptor.Decrypt(v.Secret.Value)
		if err != nil {
			return nil, err
		}

		secrets = append(secrets, &types2.Secret{
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

func (s *secretService) CreateCredentials(ctx context.Context, params types2.AddCredentialsParams) ([]*types2.Credential, error) {
	result := make([]*types2.Credential, 0)
	for _, v := range params.Values {
		credValue, err := s.encryptor.Encrypt(v.Value)
		if err != nil {
			return nil, err
		}

		cred, err := s.credentialRepository.FindByName(ctx, params.ApplicationID, params.Provider, v.Key)
		if err == nil && cred.ID != uuid.Nil {
			if err := s.credentialRepository.UpdateCredentialValue(ctx, cred.ID, credValue); err != nil {
				return nil, err
			}
		} else {
			cred = &types2.Credential{
				ID:            uuid.New(),
				ApplicationID: params.ApplicationID,
				Provider:      params.Provider,
				Name:          v.Key,
				Value:         credValue,
				CreatedAt:     time.Now(),
			}
			if err := s.credentialRepository.Save(ctx, cred); err != nil {
				return nil, err
			}
		}
		result = append(result, cred)
	}
	return result, nil
}

func (s *secretService) FindApplicationCredentials(ctx context.Context, applicationID uuid.UUID, provider string) ([]*types2.Credential, error) {
	all, err := s.credentialRepository.FindByApplicationID(ctx, applicationID)
	if err != nil {
		return nil, err
	}

	credsByProvider := lo.Filter(all, func(item *types2.Credential, index int) bool {
		return item.Provider == provider
	})

	result := make([]*types2.Credential, 0, len(credsByProvider))
	for _, next := range credsByProvider {
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

func FindSecret(name string, secrets []*types2.Secret) (*types2.Secret, error) {
	for _, next := range secrets {
		if next.Name == name {
			return next, nil
		}
	}
	return nil, fmt.Errorf("secret: %s was not found", name)
}

func FindCredential(name string, creds []*types2.Credential) (*types2.Credential, error) {
	for _, next := range creds {
		if next.Name == name {
			return next, nil
		}
	}
	return nil, fmt.Errorf("credential: %s was not found", name)
}
