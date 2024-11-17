package caddy

import (
	"context"
	"errors"
	"sarabi/logger"
	"sarabi/service"
	"sarabi/types"
	"time"
)

var (
	defaultLoadBalancerAccess = []string{":80", ":443"}
	caddyAdminAccessEndpoint  = ":2019"
	mainServer                = "main"
)

type Client interface {
	ApplyConfig(ctx context.Context, caddyUrl string, instanceType types.InstanceType, deployment *types.Deployment) error
	Wait(ctx context.Context, caddyUrl string) error
}

type caddyClient struct {
	httpClient HttpClient
	appService service.ApplicationService
}

func NewCaddyClient(appService service.ApplicationService) Client {
	return &caddyClient{httpClient: newCaddyHttpClient(), appService: appService}
}

// ApplyConfig apply configuration for a specific instance type
// it sends a request to get current caddy configuration, apply the patch and then update caddy with the new config
func (c *caddyClient) ApplyConfig(ctx context.Context, caddyUrl string, instanceType types.InstanceType, deployment *types.Deployment) error {
	switch instanceType {
	case types.InstanceTypeBackend:
		return c.patchBackendConfig(ctx, caddyUrl, deployment)
	case types.InstanceTypeFrontend:
		return c.patchFrontendConfig(ctx, caddyUrl, deployment)
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

func (c *caddyClient) patchBackendConfig(ctx context.Context, caddyUrl string, deployment *types.Deployment) error {
	cfg := &Config{}
	err := c.httpClient.Do(ctx, "GET", caddyUrl, nil, cfg)
	if err != nil {
		return err
	}

	servers := cfg.Apps.HTTP.Servers
	if servers == nil {
		servers = map[string]*Server{
			mainServer: {Listen: defaultLoadBalancerAccess},
		}
	}

	srv := servers[mainServer]
	updatedRoutes := c.removeOldRoutes(srv, deployment.AccessURL(types.InstanceTypeBackend))
	upStreams := make([]Upstream, 0, deployment.Instances)
	for idx := 0; idx < deployment.Instances; idx++ {
		upStreams = append(upStreams, Upstream{
			Dial: deployment.InternalAccessURL(idx),
		})
	}

	handles := make([]Handle, 0)
	handles = append(handles, Handle{Handler: "reverse_proxy", Upstreams: upStreams})
	updatedRoutes = append(updatedRoutes, Route{
		Handle: handles,
		Match: []Match{
			{Host: []string{deployment.AccessURL(types.InstanceTypeBackend)}},
		},
	})

	srv.Routes = updatedRoutes
	servers[mainServer] = srv
	updatedCfg := Config{
		Apps: Apps{
			HTTP: HTTP{
				Servers: servers,
			},
		},
		Admin: AdminConfig{Listen: caddyAdminAccessEndpoint},
	}
	err = c.httpClient.Do(ctx, "POST", caddyUrl, updatedCfg, nil)
	if err != nil {
		return err
	}

	if err := c.Wait(ctx, caddyUrl); err != nil {
		return err
	}
	return nil
}

func (c *caddyClient) patchFrontendConfig(ctx context.Context, caddyUrl string, deployment *types.Deployment) error {
	cfg := &Config{}
	err := c.httpClient.Do(ctx, "GET", caddyUrl, nil, cfg)
	if err != nil {
		return err
	}

	servers := cfg.Apps.HTTP.Servers
	if servers == nil {
		servers = map[string]*Server{
			mainServer: {Listen: defaultLoadBalancerAccess},
		}
	}

	srv := servers[mainServer]
	updatedRoutes := c.removeOldRoutes(srv, deployment.AccessURL(types.InstanceTypeFrontend))
	handles := make([]Handle, 0)
	handles = append(handles, Handle{Handler: "file_server", Root: deployment.SiteContentPath()})
	updatedRoutes = append(updatedRoutes, Route{
		Handle: handles,
		Match: []Match{
			{Host: []string{deployment.AccessURL(types.InstanceTypeFrontend)}},
		},
	})

	srv.Routes = updatedRoutes
	servers[mainServer] = srv
	updatedCfg := Config{
		Apps: Apps{
			HTTP: HTTP{
				Servers: servers,
			},
		},
		Admin: AdminConfig{Listen: caddyAdminAccessEndpoint},
	}

	err = c.httpClient.Do(ctx, "POST", caddyUrl, updatedCfg, nil)
	if err != nil {
		return err
	}

	if err := c.Wait(ctx, caddyUrl); err != nil {
		return err
	}
	return nil
}

func (c *caddyClient) removeOldRoutes(srv *Server, host string) []Route {
	var updatedRoutes []Route
	for _, route := range srv.Routes {
		var matches []Match
		for _, match := range route.Match {
			var filteredHosts []string
			for _, h := range match.Host {
				if h != host {
					filteredHosts = append(filteredHosts, h)
				}
			}

			// If any hosts remain after filtering, include this match
			if len(filteredHosts) > 0 {
				matches = append(matches, Match{
					Host: filteredHosts,
				})
			}
		}

		// Only include the route if it has valid matches remaining
		if len(matches) > 0 {
			updatedRoutes = append(updatedRoutes, Route{
				Handle: route.Handle,
				Match:  matches,
			})
		}
	}

	return updatedRoutes
}
