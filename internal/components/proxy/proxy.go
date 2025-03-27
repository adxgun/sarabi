package proxycomponent

import (
	"context"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"sarabi/internal/components"
	"sarabi/internal/integrations/caddy"
	"sarabi/internal/integrations/docker"
	"sarabi/internal/service"
)

var (
	caddyImageName               = "adxgun/sarabi-caddy:2.9-v3"
	ProxyServerName              = "main-proxy-server"
	proxyStaticFilesVolume       = "sarabi-statics"
	proxyConfigVolume            = "sarabi-proxy-config"
	defaultCaddyApiConfigContent = `
		{
			admin 127.0.0.1:2019
		}
		`
)

type (
	proxyComponent struct {
		dockerClient docker.Docker
		appService   service.ApplicationService
		caddyClient  caddy.Client
	}
)

func New(
	dockerClient docker.Docker,
	appService service.ApplicationService,
	caddyClient caddy.Client) components.Builder {
	return &proxyComponent{
		dockerClient: dockerClient,
		appService:   appService,
		caddyClient:  caddyClient,
	}
}

func (p *proxyComponent) Name() string {
	return "proxy"
}

func (p *proxyComponent) Run(ctx context.Context, deploymentID uuid.UUID) (*components.BuilderResult, error) {
	running, info, err := p.dockerClient.IsContainerRunning(ctx, ProxyServerName)
	if err == nil && running {
		return &components.BuilderResult{
			ID:   info.ID,
			Name: info.Name,
		}, nil
	}

	err = p.dockerClient.CreateVolume(ctx, proxyStaticFilesVolume)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create static files volume")
	}

	err = p.dockerClient.CreateVolume(ctx, proxyConfigVolume)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create config volume")
	}

	err = p.dockerClient.PullImage(ctx, caddyImageName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to pull caddy image")
	}

	// open http, https and the caddy API port
	httpPort, _ := nat.NewPort("tcp", "80")
	httpsPort, _ := nat.NewPort("tcp", "443")
	apiPort, _ := nat.NewPort("tcp", "2019")
	exposedPorts := []nat.Port{httpPort, httpsPort, apiPort}
	portBindings := nat.PortMap{
		httpPort:  []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "80"}},
		httpsPort: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "443"}},
		apiPort:   []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: "2019"}},
	}

	mounts := map[string]string{
		proxyConfigVolume: "/caddy_data",
	}
	volumes := []string{
		"/var/caddy/share/:/var/caddy/share/",
	}

	params := docker.StartContainerParams{
		Image:        caddyImageName,
		Container:    ProxyServerName,
		ExposedPorts: exposedPorts,
		PortBindings: portBindings,
		Mounts:       mounts,
		Cmd: []string{
			"caddy", "run", "--config", "/etc/caddy/caddy.json", "--resume",
		},
		Volumes: volumes,
	}
	result, err := p.dockerClient.StartContainerAndWait(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to start caddy container")
	}

	if err := p.caddyClient.Wait(ctx); err != nil {
		return nil, errors.Wrap(err, "caddy failed to start")
	}

	// TODO: init caddy based on its saved state
	if err := p.caddyClient.Init(ctx); err != nil {
		return nil, errors.Wrap(err, "caddy failed to init")
	}

	return &components.BuilderResult{
		ID:   result.ID,
		Name: result.Name,
	}, nil
}

func (p *proxyComponent) Cleanup(ctx context.Context, result *components.BuilderResult) error {
	return p.dockerClient.StopAndRemoveContainer(ctx, docker.StopContainerParams{
		RemoveVolumes: false,
		ContainerName: result.ID,
	})
}
