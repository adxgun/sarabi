package manager

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"io"
	"os"
	"path/filepath"
	"sarabi"
	"sarabi/bundler"
	backendcomponent "sarabi/components/backend"
	databasecomponent "sarabi/components/database"
	frontendcomponent "sarabi/components/frontend"
	"sarabi/integrations/caddy"
	"sarabi/integrations/docker"
	"sarabi/logger"
	"sarabi/service"
	"sarabi/types"
)

var (
	Path              = "/var/sarabi/data"
	DBDir             = Path + "/sarabi.db"
	ServerCrtFilePath = Path + "/certs/server.crt"
	ServerKeyFilePath = Path + "/certs/server.key"
)

type (
	Manager interface {
		ValidateToken(ctx context.Context, token string) error
		CreateApplication(ctx context.Context, param types.CreateApplicationParams) (*types.Application, error)
		Deploy(ctx context.Context, param *types.DeployParams) ([]*types.Deployment, error)
		CreateSecrets(ctx context.Context, applicationID uuid.UUID, environment string, params ...types.CreateSecretParams) error
	}
)

type manager struct {
	appService    service.ApplicationService
	secretService service.SecretService
	dockerClient  docker.Docker
	caddyClient   caddy.Client
	store         bundler.ArtifactStore
}

func New(applicationService service.ApplicationService, secretService service.SecretService,
	dockerClient docker.Docker, caddyClient caddy.Client, st bundler.ArtifactStore) Manager {
	return &manager{
		appService:    applicationService,
		secretService: secretService,
		dockerClient:  dockerClient,
		caddyClient:   caddyClient,
		store:         st,
	}
}

func (m *manager) ValidateToken(ctx context.Context, token string) error {
	tokenPath := filepath.Join(Path, "auth.secure")
	fi, err := os.Open(tokenPath)
	if err != nil {
		return err
	}

	content, err := io.ReadAll(fi)
	if err != nil {
		return err
	}

	if string(content) != token {
		return errors.New("access denied")
	}
	return nil
}

func (m *manager) CreateApplication(ctx context.Context, param types.CreateApplicationParams) (*types.Application, error) {
	return m.appService.Create(ctx, param)
}

func (m *manager) Deploy(ctx context.Context, param *types.DeployParams) ([]*types.Deployment, error) {
	var backendDeployment *types.Deployment
	var frontendDeployment *types.Deployment
	var deployments = make([]*types.Deployment, 0)
	identifier, err := sarabi.DefaultRandomIdGenerator.Generate(10)
	if err != nil {
		return nil, err
	}

	if param.Backend != nil {
		dbDeployment, err := m.appService.CreateDeployment(ctx, types.CreateDeploymentParams{
			ApplicationID: param.ApplicationID,
			Environment:   param.Environment,
			InstanceType:  types.InstanceTypeDatabase,
			Identifier:    identifier,
		})
		if err != nil {
			return nil, err
		}

		dbComponent := databasecomponent.New(m.dockerClient, m.appService,
			m.secretService, databasecomponent.NewProvider(param.StorageEngine))
		if _, err := dbComponent.Run(ctx, dbDeployment.ID); err != nil {
			return nil, err
		}

		appPort, err := sarabi.DefaultPortGenerator.Generate()
		if err != nil {
			return nil, err
		}

		createBackend := types.CreateDeploymentParams{
			ApplicationID: param.ApplicationID,
			Environment:   param.Environment,
			Instances:     param.Instances,
			Port:          appPort,
			InstanceType:  types.InstanceTypeBackend,
			Identifier:    identifier,
		}
		backendDeployment, err = m.appService.CreateDeployment(ctx, createBackend)
		if err != nil {
			return nil, err
		}

		if err := m.store.Save(ctx, param.Backend, backendDeployment); err != nil {
			return nil, err
		}

		if err := m.setupAppSecrets(ctx, backendDeployment); err != nil {
			return nil, err
		}

		deployments = append(deployments, backendDeployment)
	}

	if param.Frontend != nil {
		createFrontend := types.CreateDeploymentParams{
			ApplicationID: param.ApplicationID,
			Environment:   param.Environment,
			Instances:     param.Instances,
			InstanceType:  types.InstanceTypeBackend,
			Identifier:    identifier,
		}
		fd, err := m.appService.CreateDeployment(ctx, createFrontend)
		if err != nil {
			return nil, err
		}
		if err := m.store.Save(ctx, param.Frontend, fd); err != nil {
			return nil, err
		}
		frontendDeployment = fd
		deployments = append(deployments, frontendDeployment)
	}

	if backendDeployment != nil {
		backend := backendcomponent.New(m.dockerClient, m.appService, m.secretService, m.caddyClient)
		result, err := backend.Run(ctx, backendDeployment.ID)
		if err != nil {
			return nil, err
		}

		if err := backend.Cleanup(ctx, result); err != nil {
			logger.Warn("cleanup failed: ", zap.Error(err))
		}
	}

	if frontendDeployment != nil {
		frontend := frontendcomponent.New(m.dockerClient, m.appService, m.secretService, m.caddyClient)
		result, err := frontend.Run(ctx, frontendDeployment.ID)
		if err != nil {
			return nil, err
		}
		if err := frontend.Cleanup(ctx, result); err != nil {
			return nil, err
		}
	}
	return deployments, nil
}

func (m *manager) CreateSecrets(ctx context.Context, applicationID uuid.UUID, environment string, params ...types.CreateSecretParams) error {
	logger.Info("new request",
		zap.Any("paras", params),
		zap.String("application_id", applicationID.String()),
		zap.String("env", environment))

	identifier, err := sarabi.DefaultRandomIdGenerator.Generate(10)
	if err != nil {
		return err
	}
	activeBackendDeployment, err := m.appService.FindCurrentlyActiveDeploymentsEnv(ctx, applicationID, types.InstanceTypeBackend, environment)
	if err != nil {
		return err
	}

	newBackendDeployment, err := m.appService.CreateDeployment(ctx, types.CreateDeploymentParams{
		ApplicationID: applicationID,
		Environment:   environment,
		Instances:     activeBackendDeployment.Instances,
		Port:          activeBackendDeployment.Port,
		InstanceType:  types.InstanceTypeBackend,
		Identifier:    identifier,
	})
	if err != nil {
		return err
	}

	if err := m.store.Copy(ctx, activeBackendDeployment, newBackendDeployment); err != nil {
		return err
	}

	oldVars, err := m.secretService.FindDeploymentSecrets(ctx, activeBackendDeployment.ID)
	if err != nil {
		return err
	}

	newVars := make([]types.CreateSecretParams, 0, len(params))
	for _, nextParam := range params {
		newVars = append(newVars, types.CreateSecretParams{
			Key:           nextParam.Key,
			Value:         nextParam.Value,
			Environment:   environment,
			InstanceType:  types.InstanceTypeBackend,
			ApplicationID: applicationID,
		})
	}

	mergedVars := m.mergeSecrets(oldVars, newVars)
	createdVars, err := m.secretService.CreateAll(ctx, mergedVars...)
	if err != nil {
		return err
	}

	err = m.secretService.CreateDeploymentSecrets(ctx, newBackendDeployment.ID, createdVars)
	if err != nil {
		return err
	}

	backend := backendcomponent.New(m.dockerClient, m.appService, m.secretService, m.caddyClient)
	r, err := backend.Run(ctx, newBackendDeployment.ID)
	if err != nil {
		return err
	}
	if err := backend.Cleanup(ctx, r); err != nil {
		logger.Warn("backend cleanup error: ", zap.Error(err))
	}

	return nil
}

func (m *manager) mergeSecrets(oldVars []*types.Secret, newVars []types.CreateSecretParams) []types.CreateSecretParams {
	var mergedSecrets = append([]types.CreateSecretParams{}, newVars...)
	for _, nextOldVar := range oldVars {
		found := false
		for _, nextNewVar := range newVars {
			if nextNewVar.Key == nextOldVar.Name {
				found = true
				break
			}
		}
		if !found {
			mergedSecrets = append(mergedSecrets, types.CreateSecretParams{
				Key:           nextOldVar.Name,
				Value:         nextOldVar.Value,
				Environment:   nextOldVar.Environment,
				InstanceType:  types.InstanceType(nextOldVar.InstanceType),
				ApplicationID: nextOldVar.ApplicationID,
			})
		}
	}
	return mergedSecrets
}

func (m *manager) setupAppSecrets(ctx context.Context, deployment *types.Deployment) error {
	secret, err := m.secretService.Create(ctx, types.CreateSecretParams{
		Key:           "PORT",
		Value:         deployment.Port,
		Environment:   deployment.Environment,
		InstanceType:  deployment.InstanceType,
		ApplicationID: deployment.ApplicationID,
	})
	if err != nil {
		return err
	}

	appSecrets, err := m.secretService.FindAll(ctx, deployment.ApplicationID)
	if err != nil {
		return err
	}

	dbSecrets := make([]*types.Secret, 0)
	for _, ss := range appSecrets {
		if types.InstanceType(ss.InstanceType) == types.InstanceTypeDatabase &&
			ss.Environment == deployment.Environment {
			dbSecrets = append(dbSecrets, ss)
		}
	}

	dbSecrets = append(dbSecrets, secret)
	return m.secretService.CreateDeploymentSecrets(ctx, deployment.ID, dbSecrets)
}
