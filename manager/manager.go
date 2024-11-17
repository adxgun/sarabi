package manager

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"io"
	"os"
	"path/filepath"
	"sarabi"
	"sarabi/components"
	backendcomponent "sarabi/components/backend"
	databasecomponent "sarabi/components/database"
	frontendcomponent "sarabi/components/frontend"
	proxycomponent "sarabi/components/proxy"
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
		CreateSecrets(ctx context.Context, params ...types.CreateSecretParams) error
	}
)

type manager struct {
	appService    service.ApplicationService
	secretService service.SecretService
	dockerClient  docker.Docker
	caddyClient   caddy.Client
}

func New(applicationService service.ApplicationService, secretService service.SecretService,
	dockerClient docker.Docker, caddyClient caddy.Client) Manager {
	return &manager{
		appService:    applicationService,
		secretService: secretService,
		dockerClient:  dockerClient,
		caddyClient:   caddyClient,
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

	if param.Backend != nil {
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
		}
		backendDeployment, err = m.appService.CreateDeployment(ctx, createBackend)
		if err != nil {
			return nil, err
		}

		if err := m.saveDeploymentBin(param.Backend, backendDeployment); err != nil {
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
		}
		fd, err := m.appService.CreateDeployment(ctx, createFrontend)
		if err != nil {
			return nil, err
		}
		if err := m.saveDeploymentBin(param.Frontend, fd); err != nil {
			return nil, err
		}
		frontendDeployment = fd
		deployments = append(deployments, frontendDeployment)
	}

	dbAndProxy := make([]components.Builder, 0, 2)
	dbAndProxy = append(dbAndProxy, proxycomponent.New(m.dockerClient, m.appService, m.caddyClient))
	dbAndProxy = append(dbAndProxy, databasecomponent.New(m.dockerClient, m.appService, m.secretService))

	if backendDeployment != nil {
		for _, dp := range dbAndProxy {
			rs, err := dp.Run(ctx, backendDeployment.ID)
			if err != nil {
				return nil, err
			}
			if err := dp.Cleanup(ctx, rs); err != nil {
				logger.Warn("cleanup failed: ", zap.Error(err))
			}
		}

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

func (m *manager) CreateSecrets(ctx context.Context, params ...types.CreateSecretParams) error {
	return nil
}

// saveDeploymentBin writes the deployment binary to the host file system
// the purpose of this is to enable support for deployment rollback.
// there is a different provision for purging out the binaries once they're old(60 days by default)
func (m *manager) saveDeploymentBin(r io.Reader, deployment *types.Deployment) error {
	logger.Info("saving bin: ", zap.String("path", deployment.BinPath()))
	dir := filepath.Dir(deployment.BinPath())
	logger.Info("saving bin: ", zap.String("path", dir))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	fi, err := os.Create(deployment.BinPath())
	if err != nil {
		return err
	}

	logger.Info("bin file created: ", zap.String("name", deployment.BinPath()))
	defer fi.Close()
	_, err = io.Copy(fi, r)
	if err != nil {
		return err
	}

	st, err := os.Stat(deployment.BinPath())
	logger.Info("err", zap.Error(err))
	logger.Info("data",
		zap.String("name", st.Name()),
		zap.Int64("size", st.Size()),
		zap.Bool("is_dir", st.IsDir()),
		zap.Any("mode", st.Mode()),
		zap.Any("mod_time", st.ModTime()))

	return nil
}

func (m *manager) setupAppSecrets(ctx context.Context, deployment *types.Deployment) error {
	secrets := types.DBSecrets(deployment)
	secrets = append(secrets, types.CreateSecretParams{
		Key:           "PORT",
		Value:         deployment.Port,
		Environment:   deployment.Environment,
		InstanceType:  deployment.InstanceType,
		ApplicationID: deployment.ApplicationID,
	})
	secret, err := m.secretService.CreateAll(ctx, secrets...)
	if err != nil {
		return err
	}

	appSecrets, err := m.secretService.FindAll(ctx, deployment.ApplicationID)
	if err != nil {
		return err
	}

	appSecrets = append(appSecrets, secret...)
	return m.secretService.CreateDeploymentSecrets(ctx, deployment.ID, appSecrets)
}
