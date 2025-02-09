package frontendcomponent

import (
	"context"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"sarabi/internal/bundler"
	"sarabi/internal/components"
	"sarabi/internal/eventbus"
	"sarabi/internal/integrations/caddy"
	"sarabi/internal/integrations/docker"
	"sarabi/internal/service"
	"sarabi/internal/types"
	"sarabi/logger"
)

type (
	frontendComponent struct {
		dockerClient  docker.Docker
		appService    service.ApplicationService
		secretService service.SecretService
		caddyClient   caddy.Client
		eb            eventbus.Bus
	}
)

func New(
	dockerClient docker.Docker,
	appService service.ApplicationService,
	secretService service.SecretService,
	caddyClient caddy.Client,
	eb eventbus.Bus) components.Builder {
	return &frontendComponent{
		dockerClient:  dockerClient,
		appService:    appService,
		secretService: secretService,
		caddyClient:   caddyClient,
		eb:            eb,
	}
}

func (f *frontendComponent) Name() string {
	return string(types.InstanceTypeFrontend)
}

func (f *frontendComponent) Run(ctx context.Context, deploymentID uuid.UUID) (*components.BuilderResult, error) {
	deployment, err := f.appService.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, err
	}

	f.eb.Broadcast(deployment.Identifier, eventbus.Info, "Deploying frontend...")

	previousActives, err := f.appService.FindCurrentlyActiveDeployments(ctx, deployment.ApplicationID, types.InstanceTypeFrontend)
	if err != nil {
		return nil, err
	}

	if err := bundler.Extract(deployment.BinPath(), deployment.SiteContentPath()); err != nil {
		return nil, err
	}

	err = f.caddyClient.ApplyConfig(ctx, types.InstanceTypeFrontend, deployment)
	if err != nil {
		return nil, err
	}

	err = f.appService.UpdateDeploymentStatus(ctx, deploymentID, types.DeploymentStatusActive)
	if err != nil {
		return nil, err
	}

	f.eb.Broadcast(deployment.Identifier, eventbus.Success, "frontend deployment completed...")
	return &components.BuilderResult{
		PreviousActive: previousActives,
	}, nil
}

func (f *frontendComponent) Cleanup(ctx context.Context, result *components.BuilderResult) error {
	if result == nil || len(result.PreviousActive) == 0 {
		return nil
	}

	for _, p := range result.PreviousActive {
		err := f.appService.UpdateDeploymentStatus(ctx, p.ID, types.DeploymentStatusStopped)
		if err != nil {
			logger.Warn("failed to update deployment status",
				zap.Error(err), zap.String("component", f.Name()))
		}
	}
	return nil
}
