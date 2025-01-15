package backendcomponent

import (
	"context"
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"sarabi/internal/components"
	proxycomponent "sarabi/internal/components/proxy"
	"sarabi/internal/eventbus"
	"sarabi/internal/integrations/caddy"
	"sarabi/internal/integrations/docker"
	"sarabi/internal/service"
	"sarabi/internal/types"
	"sarabi/logger"
)

type (
	backendComponent struct {
		dockerClient  docker.Docker
		appService    service.ApplicationService
		secretService service.SecretService
		caddyClient   caddy.Client
		eb            eventbus.Bus
	}
)

func New(dc docker.Docker,
	appService service.ApplicationService,
	sc service.SecretService,
	caddyClient caddy.Client,
	eb eventbus.Bus) components.Builder {
	return &backendComponent{
		dockerClient:  dc,
		appService:    appService,
		secretService: sc,
		caddyClient:   caddyClient,
		eb:            eb,
	}
}

func (b *backendComponent) Name() string {
	return "backend"
}

func (b *backendComponent) Run(ctx context.Context, deploymentID uuid.UUID) (*components.BuilderResult, error) {
	deployment, err := b.appService.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, err
	}

	secrets, err := b.secretService.FindDeploymentSecrets(ctx, deploymentID)
	if err != nil {
		return nil, err
	}

	currentlyActives, err := b.appService.FindCurrentlyActiveDeployments(ctx, deployment.ApplicationID, types.InstanceTypeBackend)
	if err != nil {
		return nil, err
	}

	_, err = b.dockerClient.BuildImage(ctx, deployment)
	if err != nil {
		return nil, err
	}

	err = b.dockerClient.CreateNetwork(ctx, deployment.NetworkName())
	if err != nil {
		return nil, err
	}

	var envs []string
	for _, ss := range secrets {
		envs = append(envs, ss.Env())
	}

	g, ctx := errgroup.WithContext(ctx)
	for idx := 0; idx < deployment.Instances; idx++ {
		g.Go(func() error {
			b.eb.Broadcast(deployment.Identifier, eventbus.Info, fmt.Sprintf("Starting application container: replicaID=%d", idx+1))
			httpPort, _ := nat.NewPort("tcp", deployment.Port)
			networkName := deployment.NetworkName()
			params := docker.StartContainerParams{
				Image:        deployment.ImageName(),
				Container:    deployment.ContainerName(idx),
				Network:      &networkName,
				Volumes:      []string{},
				Environments: envs,
				ExposedPorts: []nat.Port{httpPort},
			}
			newInfo, err := b.dockerClient.StartContainerAndWait(ctx, params)
			if err != nil {
				return err
			}

			err = b.appService.UpdateDeploymentStatus(ctx, deploymentID, types.DeploymentStatusActive)
			if err != nil {
				return err
			}
			logger.Info("started instance",
				zap.Int("index", idx),
				zap.String("component", b.Name()),
				zap.Any("result", newInfo))
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		b.eb.Broadcast(deployment.Identifier, eventbus.Error, "Failed to start application container: "+err.Error())
		return nil, err
	}

	err = b.caddyClient.ApplyConfig(context.Background(), types.InstanceTypeBackend, deployment)
	if err != nil {
		return nil, err
	}

	if err := b.dockerClient.ConnectContainer(context.Background(), proxycomponent.ProxyServerName, deployment.NetworkName()); err != nil {
		logger.Warn("container connection error: ", zap.Error(err))
	}

	return &components.BuilderResult{
		Name:           deployment.Application.Name,
		PreviousActive: currentlyActives,
	}, nil
}

func (b *backendComponent) Cleanup(ctx context.Context, result *components.BuilderResult) error {
	if result == nil || len(result.PreviousActive) == 0 {
		return nil
	}

	for _, deployment := range result.PreviousActive {
		for idx := 0; idx < deployment.Instances; idx++ {
			err := b.dockerClient.StopAndRemoveContainer(ctx, docker.StopContainerParams{
				RemoveVolumes: true,
				ContainerName: deployment.ContainerName(idx),
			})
			if err != nil {
				return err
			}
		}
	}

	for _, deployment := range result.PreviousActive {
		err := b.appService.UpdateDeploymentStatus(ctx, deployment.ID, types.DeploymentStatusStopped)
		if err != nil {
			return err
		}
	}

	return nil
}
