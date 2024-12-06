package manager

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	errors2 "github.com/pkg/errors"
	"github.com/samber/lo"
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
	"sarabi/storage"
	"sarabi/types"
	"sort"
	"strings"
	"time"
)

const (
	defaultBackupInterval = time.Minute * 10 // 30 mins
)

type (
	Manager interface {
		ValidateToken(ctx context.Context, token string) error
		CreateApplication(ctx context.Context, param types.CreateApplicationParams) (*types.Application, error)
		Deploy(ctx context.Context, param *types.DeployParams) (*types.DeployResponse, error)
		Destroy(ctx context.Context, applicationID uuid.UUID, environment string) error
		UpdateVariables(ctx context.Context, applicationID uuid.UUID, environment string, params ...types.CreateSecretParams) error
		Rollback(ctx context.Context, identifier string) ([]*types.Deployment, error)
		Scale(ctx context.Context, applicationID uuid.UUID, newInstanceCount int) ([]*types.Deployment, error)
		AddDomain(ctx context.Context, applicationID uuid.UUID, params types.AddDomainParams) (*types.Domain, error)
		RemoveDomain(ctx context.Context, applicationID uuid.UUID, name string) error
		AddCredentials(ctx context.Context, params types.AddCredentialsParams) ([]*types.AddCredentialsResponse, error)
		DownloadBackup(ctx context.Context, backupID uuid.UUID) (*types.File, error)
		ListBackups(ctx context.Context, applicationID uuid.UUID) ([]*types.Backup, error)
		ListDeployments(ctx context.Context, applicationID uuid.UUID) ([]*types.Deployment, error)
		ListApplications(ctx context.Context) ([]*types.Application, error)
		EnableDatabaseAccess(ctx context.Context, applicationID uuid.UUID, environment, ip string) error
	}
)

type manager struct {
	appService    service.ApplicationService
	secretService service.SecretService
	dockerClient  docker.Docker
	caddyClient   caddy.Client
	store         bundler.ArtifactStore
	domainService service.DomainService
	backupService service.BackupService
}

func New(applicationService service.ApplicationService, secretService service.SecretService,
	dockerClient docker.Docker, caddyClient caddy.Client,
	st bundler.ArtifactStore, dms service.DomainService, backup service.BackupService) Manager {
	return &manager{
		appService:    applicationService,
		secretService: secretService,
		dockerClient:  dockerClient,
		caddyClient:   caddyClient,
		store:         st,
		domainService: dms,
		backupService: backup,
	}
}

func (m *manager) ValidateToken(ctx context.Context, token string) error {
	tokenPath := filepath.Join(storage.Path, "auth.secure")
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

func (m *manager) Deploy(ctx context.Context, param *types.DeployParams) (*types.DeployResponse, error) {
	var backendDeployment *types.Deployment
	var frontendDeployment *types.Deployment
	var feDomains []string
	var beDomains []string

	app, err := m.appService.Get(ctx, param.ApplicationID)
	if err != nil {
		return nil, err
	}

	identifier, err := sarabi.DefaultRandomIdGenerator.Generate(10)
	if err != nil {
		return nil, err
	}

	if param.Backend != nil {
		dbPort, err := sarabi.DefaultPortGenerator.Generate()
		if err != nil {
			return nil, err
		}

		dbDeployment, err := m.appService.CreateDeployment(ctx, types.CreateDeploymentParams{
			ApplicationID: param.ApplicationID,
			Environment:   param.Environment,
			InstanceType:  types.InstanceTypeDatabase,
			Identifier:    identifier,
			Port:          dbPort,
		})
		if err != nil {
			return nil, errors2.Wrap(err, "failed to create database deployment")
		}

		for _, se := range app.StorageEngines {
			dbComponent := databasecomponent.New(m.dockerClient, m.appService,
				m.secretService, databasecomponent.NewProvider(se), m.caddyClient)
			if _, err := dbComponent.Run(ctx, dbDeployment.ID); err != nil {
				return nil, errors2.Wrap(err, "failed to run database component")
			}
		}

		if err := m.backupService.CreateBackupSettings(ctx, param.ApplicationID, param.Environment, defaultBackupInterval); err != nil {
			return nil, errors2.Wrap(err, "failed to initialize auto-backup")
		}

		appPort, err := sarabi.DefaultPortGenerator.Generate()
		if err != nil {
			return nil, errors2.Wrap(err, "failed to allocate port")
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
			return nil, errors2.Wrap(err, "failed to save backend deployment")
		}

		if err := m.store.Save(ctx, param.Backend, backendDeployment); err != nil {
			return nil, errors2.Wrap(err, "failed to save backend artifact")
		}

		if err := m.setupAppVariables(ctx, backendDeployment); err != nil {
			return nil, errors2.Wrap(err, "failed to setup app variables")
		}
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
			return nil, errors2.Wrap(err, "failed to create frontend deployment")
		}
		if err := m.store.Save(ctx, param.Frontend, fd); err != nil {
			return nil, errors2.Wrap(err, "failed to save frontend artifact")
		}
		frontendDeployment = fd
	}

	if backendDeployment != nil {
		backend := backendcomponent.New(m.dockerClient, m.appService, m.secretService, m.caddyClient)
		result, err := backend.Run(ctx, backendDeployment.ID)
		if err != nil {
			return nil, errors2.Wrap(err, "failed to run backend component")
		}

		if err := backend.Cleanup(ctx, result); err != nil {
			logger.Warn("cleanup failed: ", zap.Error(err))
		}
		beDomains = append(beDomains, m.toURL(backendDeployment.AccessURL(types.InstanceTypeBackend)))
	}

	if frontendDeployment != nil {
		frontend := frontendcomponent.New(m.dockerClient, m.appService, m.secretService, m.caddyClient)
		result, err := frontend.Run(ctx, frontendDeployment.ID)
		if err != nil {
			return nil, errors2.Wrap(err, "failed to frontend component")
		}
		if err := frontend.Cleanup(ctx, result); err != nil {
			logger.Warn("cleanup failed: ", zap.Error(err))
		}
		feDomains = append(feDomains, m.toURL(frontendDeployment.AccessURL(types.InstanceTypeFrontend)))
	}

	domains, err := m.domainService.FindByApplicationID(ctx, param.ApplicationID)
	if err != nil {
		return nil, err
	}

	for _, do := range domains {
		if do.InstanceType == types.InstanceTypeFrontend && do.Environment == param.Environment {
			feDomains = append(feDomains, m.toURL(do.Name))
		}
		if do.InstanceType == types.InstanceTypeBackend && do.Environment == param.Environment {
			beDomains = append(beDomains, m.toURL(do.Name))
		}
	}

	return &types.DeployResponse{
		Identifier: identifier,
		AccessURL: types.AccessURL{
			Frontend: feDomains,
			Backend:  beDomains,
		},
	}, nil
}

func (m *manager) UpdateVariables(ctx context.Context, applicationID uuid.UUID, environment string, params ...types.CreateSecretParams) error {
	logger.Info("new update var request",
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

func (m *manager) Rollback(ctx context.Context, identifier string) ([]*types.Deployment, error) {
	deployments, err := m.appService.FindDeploymentsByIdentifier(ctx, identifier)
	if err != nil {
		return nil, err
	}

	newIdentifier, err := sarabi.DefaultRandomIdGenerator.Generate(10)
	if err != nil {
		return nil, err
	}

	result := make([]*types.Deployment, 0)
	var beDeployment *types.Deployment
	var feDeployment *types.Deployment

	be := lo.Filter(deployments, func(item *types.Deployment, index int) bool {
		return item.InstanceType == types.InstanceTypeBackend
	})
	fe := lo.Filter(deployments, func(item *types.Deployment, index int) bool {
		return item.InstanceType == types.InstanceTypeFrontend
	})
	if len(be) > 0 {
		beDeployment = be[0]
	}
	if len(fe) > 0 {
		feDeployment = fe[0]
	}

	if beDeployment != nil {
		vars, err := m.secretService.FindDeploymentSecrets(ctx, beDeployment.ID)
		if err != nil {
			return nil, err
		}
		createVarsParams := lo.Map(vars, func(item *types.Secret, index int) types.CreateSecretParams {
			return types.CreateSecretParams{
				Key:           item.Name,
				Value:         item.Value,
				ApplicationID: item.ApplicationID,
				Environment:   item.Environment,
				InstanceType:  types.InstanceType(item.InstanceType),
			}
		})
		newBeDeployment, err := m.appService.CreateDeployment(ctx, types.CreateDeploymentParams{
			ApplicationID: beDeployment.ApplicationID,
			Environment:   beDeployment.Environment,
			Instances:     beDeployment.Instances,
			Port:          beDeployment.Port,
			InstanceType:  beDeployment.InstanceType,
			Identifier:    newIdentifier,
		})
		if err != nil {
			return nil, err
		}

		if err := m.store.Copy(ctx, beDeployment, newBeDeployment); err != nil {
			return nil, err
		}

		newVars, err := m.secretService.CreateAll(ctx, createVarsParams...)
		if err != nil {
			return nil, err
		}

		err = m.secretService.CreateDeploymentSecrets(ctx, newBeDeployment.ID, newVars)
		if err != nil {
			return nil, err
		}

		backend := backendcomponent.New(m.dockerClient, m.appService, m.secretService, m.caddyClient)
		r, err := backend.Run(ctx, newBeDeployment.ID)
		if err != nil {
			return nil, err
		}

		if err := backend.Cleanup(ctx, r); err != nil {
			logger.Warn("backend cleanup failed: ", zap.Error(err))
		}
		result = append(result, newBeDeployment)
	}

	if feDeployment != nil {
		newFeDeployment, err := m.appService.CreateDeployment(ctx, types.CreateDeploymentParams{
			ApplicationID: feDeployment.ApplicationID,
			Environment:   feDeployment.Environment,
			Instances:     feDeployment.Instances,
			Port:          feDeployment.Port,
			InstanceType:  feDeployment.InstanceType,
			Identifier:    newIdentifier,
		})
		if err != nil {
			return nil, err
		}

		if err := m.store.Copy(ctx, feDeployment, newFeDeployment); err != nil {
			return nil, err
		}

		frontend := frontendcomponent.New(m.dockerClient, m.appService, m.secretService, m.caddyClient)
		r, err := frontend.Run(ctx, newFeDeployment.ID)
		if err != nil {
			return nil, err
		}

		if err := frontend.Cleanup(ctx, r); err != nil {
			logger.Warn("frontend cleanup failed: ", zap.Error(err))
		}
		result = append(result, newFeDeployment)
	}

	return result, nil
}

func (m *manager) Scale(ctx context.Context, applicationID uuid.UUID, newInstanceCount int) ([]*types.Deployment, error) {
	deployments, err := m.appService.FindCurrentlyActiveDeployments(ctx, applicationID, types.InstanceTypeBackend)
	if err != nil {
		return nil, err
	}

	if len(deployments) == 0 {
		return nil, errors.New("no active backend deployment found")
	}

	newIdentifier, err := sarabi.DefaultRandomIdGenerator.Generate(10)
	if err != nil {
		return nil, err
	}

	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].CreatedAt.Before(deployments[j].CreatedAt)
	})

	beDeployment := deployments[0]
	vars, err := m.secretService.FindDeploymentSecrets(ctx, beDeployment.ID)
	if err != nil {
		return nil, err
	}
	createVarsParams := lo.Map(vars, func(item *types.Secret, index int) types.CreateSecretParams {
		return types.CreateSecretParams{
			Key:           item.Name,
			Value:         item.Value,
			ApplicationID: item.ApplicationID,
			Environment:   item.Environment,
			InstanceType:  types.InstanceType(item.InstanceType),
		}
	})
	newBeDeployment, err := m.appService.CreateDeployment(ctx, types.CreateDeploymentParams{
		ApplicationID: beDeployment.ApplicationID,
		Environment:   beDeployment.Environment,
		Instances:     newInstanceCount,
		Port:          beDeployment.Port,
		InstanceType:  beDeployment.InstanceType,
		Identifier:    newIdentifier,
	})
	if err != nil {
		return nil, err
	}

	if err := m.store.Copy(ctx, beDeployment, newBeDeployment); err != nil {
		return nil, err
	}

	newVars, err := m.secretService.CreateAll(ctx, createVarsParams...)
	if err != nil {
		return nil, err
	}

	err = m.secretService.CreateDeploymentSecrets(ctx, newBeDeployment.ID, newVars)
	if err != nil {
		return nil, err
	}

	backend := backendcomponent.New(m.dockerClient, m.appService, m.secretService, m.caddyClient)
	r, err := backend.Run(ctx, newBeDeployment.ID)
	if err != nil {
		return nil, err
	}

	if err := backend.Cleanup(ctx, r); err != nil {
		logger.Warn("backend cleanup failed: ", zap.Error(err))
	}

	return []*types.Deployment{newBeDeployment}, nil
}

func (m *manager) AddDomain(ctx context.Context, applicationID uuid.UUID, params types.AddDomainParams) (*types.Domain, error) {
	domain, err := m.domainService.AddDomain(ctx, applicationID, params)
	if err != nil {
		return nil, err
	}

	deployment, err := m.appService.FindCurrentlyActiveDeploymentsEnv(ctx, applicationID,
		params.InstanceType, params.Environment)
	if err != nil {
		return nil, err
	}

	err = m.caddyClient.ApplyDomainConfig(ctx, domain, deployment, types.DomainOperationAdd)
	if err != nil {
		return nil, err
	}
	return domain, nil
}

func (m *manager) RemoveDomain(ctx context.Context, applicationID uuid.UUID, name string) error {
	removed, err := m.domainService.RemoveDomain(ctx, applicationID, name)
	if err != nil {
		return err
	}

	deployment, err := m.appService.FindCurrentlyActiveDeploymentsEnv(ctx, applicationID,
		removed.InstanceType, removed.Environment)
	if err != nil {
		return err
	}

	err = m.caddyClient.ApplyDomainConfig(ctx, removed, deployment, types.DomainOperationRemove)
	if err != nil {
		return err
	}
	return nil
}

func (m *manager) AddCredentials(ctx context.Context, params types.AddCredentialsParams) ([]*types.AddCredentialsResponse, error) {
	result, err := m.secretService.CreateCredentials(ctx, params)
	if err != nil {
		return nil, err
	}

	return lo.Map(result, func(item *types.Credential, index int) *types.AddCredentialsResponse {
		return &types.AddCredentialsResponse{
			ID:       item.ID,
			Provider: item.Provider,
			Key:      item.Name,
		}
	}), nil
}

func (m *manager) DownloadBackup(ctx context.Context, backupID uuid.UUID) (*types.File, error) {
	return m.backupService.Download(ctx, backupID)
}

func (m *manager) ListBackups(ctx context.Context, applicationID uuid.UUID) ([]*types.Backup, error) {
	return m.backupService.ListBackups(ctx, applicationID)
}

func (m *manager) Destroy(ctx context.Context, applicationID uuid.UUID, environment string) error {
	logger.Info("destroying application",
		zap.String("environment", environment),
		zap.Any("applicationID", applicationID),
		zap.Bool("destroy_all?", environment == ""))

	application, err := m.appService.Get(ctx, applicationID)
	if err != nil {
		return err
	}

	allDeployments, err := m.appService.FindDeploymentsByApplication(ctx, applicationID)
	if err != nil {
		return err
	}
	var toDestroy []*types.Deployment
	switch environment {
	case "":
		toDestroy = allDeployments
	default:
		toDestroy = lo.Filter(allDeployments, func(item *types.Deployment, index int) bool {
			return item.Environment == environment
		})
	}

	backendDeployments := m.findDeploymentsByInstanceType(toDestroy, types.InstanceTypeBackend)
	frontendDeployments := m.findDeploymentsByInstanceType(toDestroy, types.InstanceTypeFrontend)
	dbDeployments := m.findDeploymentsByInstanceType(toDestroy, types.InstanceTypeDatabase)

	for _, next := range backendDeployments {
		for idx := 0; idx < next.Instances; idx++ {
			_ = m.dockerClient.StopAndRemoveContainer(ctx, next.ContainerName(idx))
		}
		err = os.Remove(next.BinPath())
		if err != nil {
			logger.Warn("failed to remove deployment bin: ", zap.Error(err))
		} else {
			logger.Info("removed deployment bin",
				zap.String("path", next.BinPath()))
		}
		// TODO: remove caddy config
		if err := m.caddyClient.RemoveConfig(ctx, next); err != nil {
			return err
		}

		err = m.secretService.DeleteDeploymentSecrets(ctx, next.ID)
		if err != nil {
			return err
		}
		err = m.appService.UpdateDeploymentStatus(ctx, next.ID, types.DeploymentStatusStopped)
		if err != nil {
			return err
		}
	}

	for _, next := range frontendDeployments {
		err = os.Remove(next.SiteContentPath())
		if err != nil {
			logger.Warn("failed to remove frontend site content for deployment",
				zap.String("deployment_id", next.ID.String()),
				zap.String("path", next.SiteContentPath()),
				zap.Error(err))
		} else {
			logger.Info("remove deployment site content",
				zap.String("path", next.SiteContentPath()))
		}

		err = m.appService.UpdateDeploymentStatus(ctx, next.ID, types.DeploymentStatusStopped)
		if err := m.caddyClient.RemoveConfig(ctx, next); err != nil {
			return err
		}
	}

	uniqueEnvs := make(map[string]bool)
	var envs []string
	for _, next := range toDestroy {
		uniqueEnvs[next.Environment] = true
	}
	for k, _ := range uniqueEnvs {
		envs = append(envs, k)
	}
	for _, se := range application.StorageEngines {
		for _, env := range envs {
			dbContainerName := fmt.Sprintf("%s-%s-%s", se, application.Name, env)
			_ = m.dockerClient.StopAndRemoveContainer(ctx, dbContainerName)
		}
	}

	for _, next := range dbDeployments {
		err = m.secretService.DeleteDeploymentSecrets(ctx, next.ID)
		if err != nil {
			return err
		}
		err = m.appService.UpdateDeploymentStatus(ctx, next.ID, types.DeploymentStatusStopped)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *manager) ListDeployments(ctx context.Context, applicationID uuid.UUID) ([]*types.Deployment, error) {
	return m.appService.FindDeploymentsByApplication(ctx, applicationID)
}

func (m *manager) ListApplications(ctx context.Context) ([]*types.Application, error) {
	return m.appService.List(ctx)
}

func (m *manager) EnableDatabaseAccess(ctx context.Context, applicationID uuid.UUID, environment, ip string) error {
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

func (m *manager) setupAppVariables(ctx context.Context, deployment *types.Deployment) error {
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

	deploymentSecrets := lo.Filter(appSecrets, func(item *types.Secret, index int) bool {
		return item.Environment == deployment.Environment
	})
	deploymentSecrets = append(deploymentSecrets, secret)
	return m.secretService.CreateDeploymentSecrets(ctx, deployment.ID, deploymentSecrets)
}

func (m *manager) toURL(s string) string {
	if strings.HasPrefix(s, "https") {
		return s
	}
	return fmt.Sprintf("https://%s", s)
}

func (m *manager) findDeploymentsByInstanceType(deps []*types.Deployment, instanceType types.InstanceType) []*types.Deployment {
	return lo.Filter(deps, func(item *types.Deployment, index int) bool {
		return item.InstanceType == instanceType
	})
}
