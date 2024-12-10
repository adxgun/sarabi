package databasecomponent

import (
	"context"
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"sarabi/internal/components"
	"sarabi/internal/integrations/caddy"
	"sarabi/internal/integrations/docker"
	service2 "sarabi/internal/service"
	"sarabi/internal/storage"
	"sarabi/internal/types"
	"sarabi/logger"
)

type databaseComponent struct {
	dockerClient  docker.Docker
	appService    service2.ApplicationService
	secretService service2.SecretService
	caddyClient   caddy.Client
	dbProvider    Provider
}

func New(dc docker.Docker, appSvc service2.ApplicationService, secretService service2.SecretService,
	dbProvider Provider, caddyClient caddy.Client) components.Builder {
	return &databaseComponent{
		dockerClient:  dc,
		appService:    appSvc,
		secretService: secretService,
		dbProvider:    dbProvider,
		caddyClient:   caddyClient,
	}
}

func (d *databaseComponent) Name() string {
	return "database:" + d.dbProvider.Image()
}

func (d *databaseComponent) Run(ctx context.Context, deploymentID uuid.UUID) (*components.BuilderResult, error) {
	logger.Info("running application component: database",
		zap.String("dockerImage", d.dbProvider.Image()))
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

	volumeMounts := []string{
		fmt.Sprintf("%s:%s", d.dbProvider.DataPath(), deployment.DatabaseMountVolume()),
		storage.BackupTempDir + ":" + storage.BackupTempDir,
	}
	tcpPort, _ := nat.NewPort("tcp", d.dbProvider.Port())
	exposedPorts := []nat.Port{tcpPort}
	portBindings := nat.PortMap{
		tcpPort: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: deployment.Port}},
	}
	params := docker.StartContainerParams{
		Image:        d.dbProvider.Image(),
		Container:    d.dbProvider.ContainerName(deployment),
		Network:      deployment.NetworkName(),
		Volumes:      volumeMounts,
		Environments: envs,
		ExposedPorts: exposedPorts,
		PortBindings: portBindings,
	}
	startResp, err := d.dockerClient.StartContainerAndWait(ctx, params)
	if err != nil {
		return nil, err
	}

	err = d.appService.UpdateDeploymentStatus(ctx, deploymentID, types.DeploymentStatusActive)
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
