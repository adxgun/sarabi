package databasecomponent

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"sarabi/components"
	"sarabi/integrations/docker"
	"sarabi/service"
)

type databaseComponent struct {
	dockerClient  docker.Docker
	appService    service.ApplicationService
	secretService service.SecretService
	dbProvider    Provider
}

func New(dc docker.Docker, appSvc service.ApplicationService, secretService service.SecretService, dbProvider Provider) components.Builder {
	return &databaseComponent{
		dockerClient:  dc,
		appService:    appSvc,
		secretService: secretService,
		dbProvider:    dbProvider,
	}
}

func (d *databaseComponent) Name() string {
	return "database:" + d.dbProvider.Image()
}

func (d *databaseComponent) Run(ctx context.Context, deploymentID uuid.UUID) (*components.BuilderResult, error) {
	deployment, err := d.appService.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, err
	}

	running, info, err := d.dockerClient.IsContainerRunning(ctx, d.dbProvider.ContainerName(deployment))
	if err != nil {
		return nil, err
	}
	if running {
		return &components.BuilderResult{ID: info.ID, Name: info.Name}, nil
	}

	dbParams := d.dbProvider.EnvVars(deployment)
	dbVars, err := d.secretService.CreateAll(ctx, dbParams...)
	if err != nil {
		return nil, err
	}

	err = d.secretService.CreateDeploymentSecrets(ctx, deploymentID, dbVars)
	if err != nil {
		return nil, err
	}

	if err := d.dbProvider.Setup(); err != nil {
		return nil, err
	}

	if err := d.dockerClient.CreateNetwork(ctx, deployment.NetworkName()); err != nil {
		return nil, err
	}

	if err := d.dockerClient.PullImage(ctx, d.dbProvider.Image()); err != nil {
		return nil, err
	}

	var envs []string
	for _, ss := range dbVars {
		envs = append(envs, ss.Env())
	}

	volumeMounts := []string{fmt.Sprintf("%s:%s", deployment.DatabaseMountVolume(), d.dbProvider.DataPath())}
	params := docker.StartContainerParams{
		Image:        d.dbProvider.Image(),
		Container:    d.dbProvider.ContainerName(deployment),
		Network:      deployment.NetworkName(),
		Volumes:      volumeMounts,
		Environments: envs,
	}
	startResp, err := d.dockerClient.StartContainerAndWait(ctx, params)
	if err != nil {
		return nil, err
	}

	return &components.BuilderResult{
		ID:   startResp.ID,
		Name: startResp.Name,
	}, nil
}

func (d *databaseComponent) Cleanup(ctx context.Context, result *components.BuilderResult) error {
	return nil
}
