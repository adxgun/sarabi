package frontendcomponent

import (
	"context"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"os"
	"sarabi/bundler"
	"sarabi/components"
	proxycomponent "sarabi/components/proxy"
	"sarabi/integrations/caddy"
	"sarabi/integrations/docker"
	"sarabi/logger"
	"sarabi/service"
	"sarabi/types"
)

type (
	frontendComponent struct {
		dockerClient  docker.Docker
		appService    service.ApplicationService
		secretService service.SecretService
		caddyClient   caddy.Client
	}
)

func New(dockerClient docker.Docker, appService service.ApplicationService,
	secretService service.SecretService, caddyClient caddy.Client) components.Builder {
	return &frontendComponent{
		dockerClient:  dockerClient,
		appService:    appService,
		secretService: secretService,
		caddyClient:   caddyClient,
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

	previousActives, err := f.appService.FindCurrentlyActiveDeployments(ctx, deployment.ApplicationID, types.InstanceTypeFrontend)
	if err != nil {
		return nil, err
	}

	if err := bundler.Extract(deployment.BinPath(), deployment.SiteContentPath()); err != nil {
		return nil, err
	}

	err = f.caddyClient.ApplyConfig(ctx, proxycomponent.ProxyServerConfigUrl, types.InstanceTypeFrontend, deployment)
	if err != nil {
		return nil, err
	}

	return &components.BuilderResult{
		PreviousActive: previousActives,
	}, nil
}

func (f *frontendComponent) Cleanup(ctx context.Context, result *components.BuilderResult) error {
	if result == nil || len(result.PreviousActive) == 0 {
		return nil
	}

	for _, p := range result.PreviousActive {
		if err := os.Remove(p.SiteContentPath()); err != nil {
			logger.Warn("failed to remove stale deployment content path",
				zap.Error(err), zap.String("component", f.Name()))
		}
	}
	return nil
}
