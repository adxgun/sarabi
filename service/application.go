package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"sarabi/database"
	"sarabi/types"
	"strings"
)

type (
	ApplicationService interface {
		Create(ctx context.Context, params types.CreateApplicationParams) (*types.Application, error)
		List(ctx context.Context) ([]*types.Application, error)
		Get(ctx context.Context, applicationID uuid.UUID) (*types.Application, error)
		GetDeployment(ctx context.Context, deploymentID uuid.UUID) (*types.Deployment, error)
		CreateDeployments(ctx context.Context, params []types.CreateDeploymentParams) ([]*types.Deployment, error)
		CreateDeployment(ctx context.Context, param types.CreateDeploymentParams) (*types.Deployment, error)
		FindCurrentlyActiveDeployments(ctx context.Context, applicationID uuid.UUID, instanceType types.InstanceType) ([]*types.Deployment, error)
		FindCurrentlyActiveDeploymentsEnv(ctx context.Context, applicationID uuid.UUID, instanceType types.InstanceType, environment string) (*types.Deployment, error)
		FindDeploymentsByIdentifier(ctx context.Context, identifier string) ([]*types.Deployment, error)
		FindDeploymentsByApplication(ctx context.Context, applicationID uuid.UUID) ([]*types.Deployment, error)
		UpdateDeploymentStatus(ctx context.Context, deploymentID uuid.UUID, status types.DeploymentStatus) error
	}
)

type applicationService struct {
	applicationRepository database.ApplicationRepository
	deploymentRepository  database.DeploymentRepository
}

func NewApplicationService(repo database.ApplicationRepository, dr database.DeploymentRepository) ApplicationService {
	return &applicationService{applicationRepository: repo, deploymentRepository: dr}
}

func (a *applicationService) Create(ctx context.Context, params types.CreateApplicationParams) (*types.Application, error) {
	if _, err := a.applicationRepository.FindByName(ctx, params.Name); err == nil {
		return nil, fmt.Errorf("application with name %s already exists", params.Name)
	}

	engine := make(types.StorageEngines, 0)
	engine = append(engine, types.StorageEngine(params.StorageEngine))
	app := &types.Application{
		ID:             uuid.New(),
		Name:           params.Name,
		Domain:         params.Domain,
		StorageEngines: engine,
	}
	if err := a.applicationRepository.Save(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}

func (a *applicationService) List(ctx context.Context) ([]*types.Application, error) {
	return a.applicationRepository.FindAll(ctx)
}

func (a *applicationService) Get(ctx context.Context, applicationID uuid.UUID) (*types.Application, error) {
	return a.applicationRepository.FindByID(ctx, applicationID)
}

func (a *applicationService) GetDeployment(ctx context.Context, deploymentID uuid.UUID) (*types.Deployment, error) {
	return a.deploymentRepository.FindByID(ctx, deploymentID)
}

func (a *applicationService) FindCurrentlyActiveDeployments(ctx context.Context, applicationID uuid.UUID, instanceType types.InstanceType) ([]*types.Deployment, error) {
	deployments, err := a.deploymentRepository.FindAll(ctx, applicationID)
	if err != nil {
		return nil, err
	}

	actives := make([]*types.Deployment, 0)
	for _, a := range deployments {
		if (a.Status == string(types.DeploymentStatusActive)) && instanceType == a.InstanceType {
			actives = append(actives, a)
		}
	}

	return actives, nil
}

func (a *applicationService) FindCurrentlyActiveDeploymentsEnv(ctx context.Context, applicationID uuid.UUID, instanceType types.InstanceType, environment string) (*types.Deployment, error) {
	actives, err := a.FindCurrentlyActiveDeployments(ctx, applicationID, instanceType)
	if err != nil {
		return nil, err
	}

	for _, next := range actives {
		if strings.ToLower(environment) == strings.ToLower(next.Environment) {
			return next, nil
		}
	}

	return nil, errors.New("no active instance found for " + string(instanceType))
}

func (a *applicationService) UpdateDeploymentStatus(ctx context.Context, deploymentID uuid.UUID, status types.DeploymentStatus) error {
	return a.deploymentRepository.UpdateDeploymentStatus(ctx, deploymentID, string(status))
}

func (a *applicationService) CreateDeployments(ctx context.Context, params []types.CreateDeploymentParams) ([]*types.Deployment, error) {
	//TODO implement me
	panic("implement me")
}

func (a *applicationService) CreateDeployment(ctx context.Context, param types.CreateDeploymentParams) (*types.Deployment, error) {
	deployment := &types.Deployment{
		ID:            uuid.New(),
		ApplicationID: param.ApplicationID,
		Environment:   param.Environment,
		Status:        "CREATED",
		Instances:     param.Instances,
		Port:          param.Port,
		InstanceType:  param.InstanceType,
		Identifier:    param.Identifier,
	}

	err := a.deploymentRepository.Save(ctx, deployment)
	if err != nil {
		return nil, err
	}

	return a.deploymentRepository.FindByID(ctx, deployment.ID)
}

func (a *applicationService) FindDeploymentsByIdentifier(ctx context.Context, identifier string) ([]*types.Deployment, error) {
	return a.deploymentRepository.FindByIdentifier(ctx, identifier)
}

func (a *applicationService) FindDeploymentsByApplication(ctx context.Context, applicationID uuid.UUID) ([]*types.Deployment, error) {
	return a.deploymentRepository.FindAll(ctx, applicationID)
}
