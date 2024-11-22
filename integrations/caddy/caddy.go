package caddy

import (
	"context"
	"errors"
	"fmt"
	"sarabi"
	"sarabi/logger"
	"sarabi/service"
	"sarabi/types"
	"time"
)

var (
	mainAccessListenPort = []string{":80", ":443"}
	caddyAdminAccessPort = ":2019"
	mainServer           = "main"
	dbProxies            = []dbProxy{
		{name: "postgresql_proxy", listen: ":5432"},
		{name: "mysql_proxy", listen: ":3306"},
	}
)

type dbProxy struct {
	name   string
	listen string
}

type Client interface {
	Init(ctx context.Context, caddyUrl string) error
	ApplyConfig(ctx context.Context, caddyUrl string, instanceType types.InstanceType, deployment *types.Deployment) error
	ApplyDomainConfig(ctx context.Context, caddyUrl string, domain *types.Domain, deployment *types.Deployment, op types.DomainOperation) error
	Wait(ctx context.Context, caddyUrl string) error
}

type caddyClient struct {
	httpClient HttpClient
	appService service.ApplicationService
}

func NewCaddyClient(appService service.ApplicationService) Client {
	return &caddyClient{httpClient: newCaddyHttpClient(), appService: appService}
}

func (c *caddyClient) Init(ctx context.Context, caddyUrl string) error {
	layer4Servers := make(map[string]*Layer4Server)
	for _, p := range dbProxies {
		layer4Servers[p.name] = &Layer4Server{
			Listen: []string{p.listen},
			Routes: make([]Layer4Route, 0),
		}
	}

	initConfig := Config{
		Apps: Apps{
			HTTP: HTTP{Servers: map[string]*Server{
				mainServer: {
					Listen: mainAccessListenPort,
					Routes: make([]Route, 0),
				},
			}},
			Layer4: Layer4{Servers: layer4Servers},
		},
		Admin: AdminConfig{
			Listen: caddyAdminAccessPort,
		},
	}

	return c.httpClient.Do(ctx, "POST", caddyUrl, initConfig, nil)
}

// ApplyConfig apply configuration for a specific instance type
// it sends a request to get current caddy configuration, apply the patch and then update caddy with the new config
func (c *caddyClient) ApplyConfig(ctx context.Context, caddyUrl string, instanceType types.InstanceType, deployment *types.Deployment) error {
	switch instanceType {
	case types.InstanceTypeBackend:
		return c.patchBackendConfig(ctx, caddyUrl, deployment)
	case types.InstanceTypeFrontend:
		return c.patchFrontendConfig(ctx, caddyUrl, deployment)
	case types.InstanceTypeDatabase:
		return c.patchDatabaseConfig(ctx, caddyUrl, deployment)
	default:
		return errors.New("instance type not supported: " + string(instanceType))
	}
}

// Wait issues a call to /config/ endpoint
// the aim is to wait until caddy is running and available to process request(s).
// featuring exponential backup and eventual failure after 10 trials
func (c *caddyClient) Wait(ctx context.Context, caddyUrl string) error {
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

func (c *caddyClient) ApplyDomainConfig(ctx context.Context, caddyUrl string, domain *types.Domain, deployment *types.Deployment, op types.DomainOperation) error {
	cfg := &Config{}
	err := c.httpClient.Do(ctx, "GET", caddyUrl, nil, cfg)
	if err != nil {
		return err
	}

	routes := cfg.Apps.HTTP.Servers[mainServer].Routes
	routeIdx := c.findRouteIndex(routes, deployment.AccessURL(domain.InstanceType))
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

func (c *caddyClient) patchBackendConfig(ctx context.Context, caddyUrl string, deployment *types.Deployment) error {
	cfg := &Config{}
	err := c.httpClient.Do(ctx, "GET", caddyUrl, nil, cfg)
	if err != nil {
		return err
	}

	// backend-stage.paas.local
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

	patchUrl := fmt.Sprintf("%sapps/http/servers/%s/routes/%d", caddyUrl, mainServer, routeIdx)
	if routeIdx == -1 {
		patchUrl := fmt.Sprintf("%sapps/http/servers/%s/routes", caddyUrl, mainServer)
		routes = append(routes, updatedRoute)
		return c.httpClient.Do(ctx, "PATCH", patchUrl, routes, nil)
	}

	return c.httpClient.Do(ctx, "PATCH", patchUrl, updatedRoute, nil)
}

func (c *caddyClient) patchFrontendConfig(ctx context.Context, caddyUrl string, deployment *types.Deployment) error {
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

func (c *caddyClient) patchDatabaseConfig(ctx context.Context, caddyUrl string, deployment *types.Deployment) error {
	cfg := &Config{}
	err := c.httpClient.Do(ctx, "GET", caddyUrl, nil, cfg)
	if err != nil {
		return err
	}

	routes := cfg.Apps.Layer4.Servers["postgresql_proxy"].Routes
	upStreams := make([]Layer4Upstream, 0)
	routeIdx := c.findLayer4RouteIndex(routes, deployment.AccessURL(types.InstanceTypeDatabase))
	dbInternalUrl := fmt.Sprintf("postgres-%s-%s:5432", deployment.Application.Name, deployment.Environment)
	// hostUrl := deployment.AccessURL(types.InstanceTypeDatabase)
	upStreams = append(upStreams, Layer4Upstream{
		Dial: []string{dbInternalUrl},
	})
	handles := make([]Layer4Handle, 0)
	handles = append(handles, Layer4Handle{Handler: "proxy", Upstreams: upStreams})
	updatedRoute := Layer4Route{
		Handle: handles,
		Match: []Layer4Match{
			{RemoteIP: Layer4RemoteIP{Ranges: []string{"192.168.1.67/24"}}},
		},
	}

	patchUrl := fmt.Sprintf("%sapps/layer4/servers/postgresql_proxy/routes/%d", caddyUrl, routeIdx)
	if routeIdx == -1 {
		patchUrl = fmt.Sprintf("%sapps/layer4/servers/postgresql_proxy/routes", caddyUrl)
		routes = append(routes, updatedRoute)
		return c.httpClient.Do(ctx, "PATCH", patchUrl, routes, nil)
	}

	return c.httpClient.Do(ctx, "PATCH", patchUrl, updatedRoute, nil)
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

func (c *caddyClient) findLayer4RouteIndex(routes []Layer4Route, host string) int {
	for idx := 0; idx < len(routes); idx++ {
		next := routes[idx]
		for _, h := range next.Match {
			if sarabi.StrContains(host, h.RemoteIP.Ranges) {
				return idx
			}
		}
	}
	return -1
}
