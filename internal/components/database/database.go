package databasecomponent

import (
	"context"
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"sarabi/internal/components"
	"sarabi/internal/eventbus"
	"sarabi/internal/integrations/caddy"
	"sarabi/internal/integrations/docker"
	service2 "sarabi/internal/service"
	"sarabi/internal/types"
	"sarabi/logger"
)

type databaseComponent struct {
	dockerClient  docker.Docker
	appService    service2.ApplicationService
	secretService service2.SecretService
	caddyClient   caddy.Client
	dbProvider    Provider
	eb            eventbus.Bus
}

func New(dc docker.Docker,
	appSvc service2.ApplicationService,
	secretService service2.SecretService,
	dbProvider Provider,
	caddyClient caddy.Client,
	eb eventbus.Bus) components.Builder {
	return &databaseComponent{
		dockerClient:  dc,
		appService:    appSvc,
		secretService: secretService,
		dbProvider:    dbProvider,
		caddyClient:   caddyClient,
		eb:            eb,
	}
}

func (d *databaseComponent) Name() string {
	return "database-" + d.dbProvider.Image()
}

func (d *databaseComponent) Run(ctx context.Context, deploymentID uuid.UUID) (*components.BuilderResult, error) {
	logger.Info("running application component: database",
		zap.String("dockerImage", d.dbProvider.Image()))

	deployment, err := d.appService.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, err
	}

	d.eb.Broadcast(deployment.Identifier, eventbus.Info, "Provisioning database: "+d.dbProvider.Image())
	running, info, err := d.dockerClient.IsContainerRunning(ctx, d.dbProvider.ContainerName(deployment))
	if err != nil {
		return nil, err
	}
	if running {
		return &components.BuilderResult{ID: info.ID, Name: info.Name}, nil
	}

	resources, err := deployment.Application.ResourcesAllocation(d.dbProvider.Engine())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get resources allocation")
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

	volumeName := fmt.Sprintf("%s-%s-%s", deployment.ApplicationID, deployment.Environment, d.dbProvider.Engine().String())
	if err := d.dockerClient.CreateVolume(ctx, volumeName); err != nil {
		return nil, err
	}

	if err := d.dockerClient.PullImage(ctx, d.dbProvider.Image()); err != nil {
		return nil, err
	}

	var envs []string
	for _, ss := range dbVars {
		envs = append(envs, ss.Env())
	}

	// TODO: Add support for health checks
	mounts := map[string]string{
		volumeName: d.dbProvider.DataPath(),
	}
	tcpPort, _ := nat.NewPort("tcp", d.dbProvider.Port())
	exposedPorts := []nat.Port{tcpPort}
	portBindings := nat.PortMap{
		tcpPort: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: deployment.Port}},
	}
	networkName := deployment.NetworkName()
	params := docker.StartContainerParams{
		Image:        d.dbProvider.Image(),
		Container:    d.dbProvider.ContainerName(deployment),
		Network:      &networkName,
		Environments: envs,
		ExposedPorts: exposedPorts,
		PortBindings: portBindings,
		Mounts:       mounts,
		Resources:    resources,
	}
	startResp, err := d.dockerClient.StartContainerAndWait(ctx, params)
	if err != nil {
		return nil, err
	}

	err = d.appService.UpdateDeploymentStatus(ctx, deploymentID, types.DeploymentStatusActive)
	if err != nil {
		return nil, err
	}

	d.eb.Broadcast(deployment.Identifier, eventbus.Success, "Database provisioning completed: "+d.dbProvider.Image())

	return &components.BuilderResult{
		ID:   startResp.ID,
		Name: startResp.Name,
	}, nil
}

func (d *databaseComponent) Cleanup(ctx context.Context, result *components.BuilderResult) error {
	return nil
}
