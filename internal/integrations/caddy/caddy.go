package caddy

import (
	"context"
	"errors"
	"fmt"
	"sarabi"
	types "sarabi/internal/types"
	"sarabi/logger"
	"time"
)

var (
	mainAccessListenPort = []string{":80", ":443"}
	caddyAdminAccessPort = ":2019"
	mainServer           = "main"
	caddyUrl             = "http://localhost:2019/config/"
)

type Client interface {
	Init(ctx context.Context) error
	ApplyConfig(ctx context.Context, instanceType types.InstanceType, deployment *types.Deployment) error
	ApplyDomainConfig(ctx context.Context, domain *types.Domain, deployment *types.Deployment, op types.DomainOperation) error
	RemoveConfig(ctx context.Context, deployment *types.Deployment) error
	Wait(ctx context.Context) error
}

type caddyClient struct {
	httpClient HttpClient
}

func NewCaddyClient() Client {
	return &caddyClient{httpClient: newCaddyHttpClient()}
}

func (c *caddyClient) Init(ctx context.Context) error {
	initConfig := Config{
		Apps: Apps{
			HTTP: HTTP{Servers: map[string]*Server{
				mainServer: {
					Listen: mainAccessListenPort,
					Routes: make([]Route, 0),
				},
			}},
		},
		Admin: AdminConfig{
			Listen: caddyAdminAccessPort,
		},
	}

	return c.httpClient.Do(ctx, "POST", caddyUrl, initConfig, nil)
}

// ApplyConfig apply configuration for a specific instance type
// it sends a request to get current caddy configuration, apply the patch and then update caddy with the new config
func (c *caddyClient) ApplyConfig(ctx context.Context, instanceType types.InstanceType, deployment *types.Deployment) error {
	switch instanceType {
	case types.InstanceTypeBackend:
		return c.patchBackendConfig(ctx, deployment, "")
	case types.InstanceTypeFrontend:
		return c.patchFrontendConfig(ctx, deployment)
	default:
		return errors.New("instance type not supported: " + string(instanceType))
	}
}

// Wait issues a call to /config/ endpoint
// the aim is to wait until caddy is running and available to process request(s).
// featuring exponential storage and eventual failure after 10 trials
func (c *caddyClient) Wait(ctx context.Context) error {
	var (
		retries = 10
		delay   = 100 * time.Millisecond
	)

	for i := 0; i < retries; i++ {
		err := c.httpClient.Do(ctx, "GET", caddyUrl, nil, nil)
		if err == nil {
			logger.Info("caddy is now available")
			return nil
		}

		time.Sleep(delay)
		delay *= 2
	}
	return errors.New("caddy failed to start")
}

func (c *caddyClient) ApplyDomainConfig(ctx context.Context, domain *types.Domain, deployment *types.Deployment, op types.DomainOperation) error {
	cfg := &Config{}
	err := c.httpClient.Do(ctx, "GET", caddyUrl, nil, cfg)
	if err != nil {
		return err
	}

	routes := cfg.Apps.HTTP.Servers[mainServer].Routes
	routeIdx := c.findRouteIndex(routes, deployment.AccessURL(domain.InstanceType))
	if routeIdx == -1 {
		return c.patchBackendConfig(ctx, deployment, domain.Name)
	}

	currentRoute := routes[routeIdx]
	hosts := make([]string, 0)
	currentHosts := currentRoute.Match[0].Host

	if op == types.DomainOperationAdd {
		hosts = append(hosts, currentHosts...)
		hosts = append(hosts, domain.Name)
	} else {
		for _, h := range currentHosts {
			if h != domain.Name {
				hosts = append(hosts, h)
			}
		}
	}

	updatedRoute := Route{
		Handle: currentRoute.Handle,
		Match:  []Match{{Host: hosts}},
	}
	patchUrl := fmt.Sprintf("%sapps/http/servers/%s/routes/%d", caddyUrl, mainServer, routeIdx)
	return c.httpClient.Do(ctx, "PATCH", patchUrl, updatedRoute, nil)
}

func (c *caddyClient) patchBackendConfig(ctx context.Context, deployment *types.Deployment, host string) error {
	cfg := &Config{}
	err := c.httpClient.Do(ctx, "GET", caddyUrl, nil, cfg)
	if err != nil {
		return err
	}

	routes := cfg.Apps.HTTP.Servers[mainServer].Routes
	routeIdx := c.findRouteIndex(routes, deployment.AccessURL(types.InstanceTypeBackend))
	upStreams := make([]Upstream, 0, deployment.Instances)
	for idx := 0; idx < deployment.Instances; idx++ {
		upStreams = append(upStreams, Upstream{
			Dial: deployment.InternalAccessURL(idx),
		})
	}

	handles := make([]Handle, 0)
	handles = append(handles, Handle{Handler: "reverse_proxy", Upstreams: upStreams})
	updatedRoute := Route{
		Handle: handles,
		Match: []Match{
			{Host: []string{deployment.AccessURL(types.InstanceTypeBackend)}},
		},
	}
	if host != "" {
		updatedRoute.Match[0].Host = append(updatedRoute.Match[0].Host, host)
	}

	patchUrl := fmt.Sprintf("%sapps/http/servers/%s/routes/%d", caddyUrl, mainServer, routeIdx)
	if routeIdx == -1 {
		patchUrl := fmt.Sprintf("%sapps/http/servers/%s/routes", caddyUrl, mainServer)
		routes = append(routes, updatedRoute)
		return c.httpClient.Do(ctx, "PATCH", patchUrl, routes, nil)
	}

	return c.httpClient.Do(ctx, "PATCH", patchUrl, updatedRoute, nil)
}

func (c *caddyClient) patchFrontendConfig(ctx context.Context, deployment *types.Deployment) error {
	cfg := &Config{}
	err := c.httpClient.Do(ctx, "GET", caddyUrl, nil, cfg)
	if err != nil {
		return err
	}

	routes := cfg.Apps.HTTP.Servers[mainServer].Routes
	routeIdx := c.findRouteIndex(routes, deployment.AccessURL(types.InstanceTypeFrontend))
	handles := make([]Handle, 0)
	handles = append(handles, Handle{Handler: "file_server", Root: deployment.SiteContentPath()})
	updatedRoute := Route{
		Handle: handles,
		Match: []Match{
			{Host: []string{deployment.AccessURL(types.InstanceTypeFrontend)}},
		},
	}

	patchUrl := fmt.Sprintf("%sapps/http/servers/%s/routes/%d", caddyUrl, mainServer, routeIdx)
	if routeIdx == -1 {
		patchUrl = fmt.Sprintf("%sapps/http/servers/%s/routes", caddyUrl, mainServer)
		routes = append(routes, updatedRoute)
		return c.httpClient.Do(ctx, "PATCH", patchUrl, routes, nil)
	}

	return c.httpClient.Do(ctx, "PATCH", patchUrl, updatedRoute, nil)
}

func (c *caddyClient) RemoveConfig(ctx context.Context, deployment *types.Deployment) error {
	cfg := &Config{}
	err := c.httpClient.Do(ctx, "GET", caddyUrl, nil, cfg)
	if err != nil {
		return err
	}

	routes := cfg.Apps.HTTP.Servers[mainServer].Routes
	routeIdx := c.findRouteIndex(routes, deployment.AccessURL(deployment.InstanceType))
	if routeIdx == -1 {
		return nil
	}

	patchUrl := fmt.Sprintf("%sapps/http/servers/%s/routes/%d", caddyUrl, mainServer, routeIdx)
	return c.httpClient.Do(ctx, "DELETE", patchUrl, nil, nil)
}

func (c *caddyClient) findRouteIndex(routes []Route, host string) int {
	for idx := 0; idx < len(routes); idx++ {
		next := routes[idx]
		for _, h := range next.Match {
			if sarabi.StrContains(host, h.Host) {
				return idx
			}
		}
	}
	return -1
}
