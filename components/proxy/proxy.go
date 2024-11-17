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
	"sarabi/types"
)

var (
	caddyImageName = "caddy:2.9"
	// allow only localhost to access caddy via API
	defaultCaddyApiConfigContent = `
		{
			admin :2019
		}
		`
	defaultConfigPath    = "/etc/caddy/%s/Caddyfile"
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
	deployment, err := p.appService.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, err
	}

	running, info := p.dockerClient.IsContainerRunning(ctx, deployment.ProxyContainerName())
	if running {
		return &components.BuilderResult{
			ID:   info.ID,
			Name: info.Name,
		}, nil
	}

	err = p.dockerClient.CreateNetwork(ctx, deployment.NetworkName())
	if err != nil {
		return nil, err
	}

	if err := p.writeCaddyInitConfig(deployment); err != nil {
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

	configPathOnHost := fmt.Sprintf(defaultConfigPath, deployment.ApplicationID)
	bindVolumes := []string{
		fmt.Sprintf("%s:/etc/caddy/Caddyfile", configPathOnHost),
		"/var/caddy/share/:/var/caddy/share",
	}
	result, err := p.dockerClient.StartContainerAndWait(ctx, caddyImageName, deployment.ProxyContainerName(),
		deployment.NetworkName(), bindVolumes, []string{}, exposedPorts, portBindings)
	if err != nil {
		return nil, err
	}

	if err := p.caddyClient.Wait(ctx, ProxyServerConfigUrl); err != nil {
		return nil, err
	}

	return &components.BuilderResult{
		ID:   result.ID,
		Name: result.Name,
	}, nil
}

func (p *proxyComponent) Cleanup(ctx context.Context, result *components.BuilderResult) error {
	return nil
}

func (p *proxyComponent) writeCaddyInitConfig(deployment *types.Deployment) error {
	path := fmt.Sprintf(defaultConfigPath, deployment.ApplicationID)
	if _, err := os.Stat(path); err == nil {
		if err := os.Remove(path); err != nil {
			return err
		}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	fi, err := os.Create(path)
	if err != nil {
		return err
	}

	_, err = io.WriteString(fi, defaultCaddyApiConfigContent)
	if err != nil {
		return err
	}
	return nil
}
