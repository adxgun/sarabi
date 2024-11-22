package proxycomponent

import (
	"context"
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"io"
	"os"
	"path/filepath"
	"sarabi/components"
	"sarabi/integrations/caddy"
	"sarabi/integrations/docker"
	"sarabi/service"
)

var (
	// caddyImageName = "caddy:2.9"
	caddyImageName = "adxgun/caddy-layer4:2.9"
	// allow only localhost to access caddy via API
	defaultCaddyApiConfigContent = `
		{
			admin :2019
		}
		`
	defaultConfigPath    = "/etc/caddy/Caddyfile"
	ProxyServerName      = "main-proxy-server"
	ProxyServerConfigUrl = "http://localhost:2019/config/"
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

	if err := p.writeCaddyInitConfig(); err != nil {
		return nil, err
	}

	err = p.dockerClient.PullImage(ctx, caddyImageName)
	if err != nil {
		return nil, err
	}

	// open http, https and the caddy API port
	httpPort, _ := nat.NewPort("tcp", "80")
	httpsPort, _ := nat.NewPort("tcp", "443")
	apiPort, _ := nat.NewPort("tcp", "2019")
	exposedPorts := []nat.Port{httpPort, httpsPort, apiPort}
	portBindings := nat.PortMap{
		httpPort:  []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "80"}},
		httpsPort: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "443"}},
		apiPort:   []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "2019"}},
	}

	bindVolumes := []string{
		fmt.Sprintf("%s:/etc/caddy/Caddyfile", defaultConfigPath),
		"/var/caddy/share/:/var/caddy/share",
	}
	params := docker.StartContainerParams{
		Image:        caddyImageName,
		Container:    ProxyServerName,
		Network:      "",
		Volumes:      bindVolumes,
		ExposedPorts: exposedPorts,
		PortBindings: portBindings,
	}
	result, err := p.dockerClient.StartContainerAndWait(ctx, params)
	if err != nil {
		return nil, err
	}

	if err := p.caddyClient.Wait(ctx, ProxyServerConfigUrl); err != nil {
		return nil, err
	}

	if err := p.caddyClient.Init(ctx, ProxyServerConfigUrl); err != nil {
		return nil, err
	}

	return &components.BuilderResult{
		ID:   result.ID,
		Name: result.Name,
	}, nil
}

func (p *proxyComponent) Cleanup(ctx context.Context, result *components.BuilderResult) error {
	return p.dockerClient.StopAndRemoveContainer(ctx, result.ID)
}

func (p *proxyComponent) writeCaddyInitConfig() error {
	if _, err := os.Stat(defaultConfigPath); err == nil {
		if err := os.Remove(defaultConfigPath); err != nil {
			return err
		}
	}

	dir := filepath.Dir(defaultConfigPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	fi, err := os.Create(defaultConfigPath)
	if err != nil {
		return err
	}

	_, err = io.WriteString(fi, defaultCaddyApiConfigContent)
	if err != nil {
		return err
	}
	return nil
}
