package docker

import (
	"context"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
	"runtime"
	"sarabi/logger"
)

var (
	dindContainerName = "sarabi-dind"
	sharedVolume      = "/var/sarabi/data/shared"
	dindImageName     = "docker:dind-rootless"
	dindPort          = "2375"
)

type (
	// Runner starts a dind(docker-in-docker) container instance on-top of the host Docker instance,
	// the purpose of this is to isolate build process(es) for deployments. Each build happens inside dind
	// away from the host docker, once build is done, the built image is saved in a shared directory where
	// it will be loaded and run by the host Docker instance.
	Runner interface {
		Run(ctx context.Context) error
	}
)

type runner struct {
	hostClient *client.Client
}

func newRunner(cli *client.Client) Runner {
	return &runner{hostClient: cli}
}

func (r runner) Run(ctx context.Context) error {
	if runtime.GOOS != "linux" {
		logger.Info("dind is only supported on linux. Will use host Docker to build and run containers",
			zap.String("current_os", runtime.GOOS))
		return nil
	}

	img, err := r.hostClient.ContainerInspect(ctx, dindContainerName)
	if err != nil {
		if !client.IsErrNotFound(err) {
			return err
		}
	}

	if err == nil && img.State != nil {
		if img.State.Running || img.State.Restarting {
			logger.Info("dind is already running")
			return nil
		}
	}

	result, err := r.hostClient.ImagePull(ctx, dindImageName, image.PullOptions{})
	if err != nil {
		return err
	}

	defer result.Close()
	port, _ := nat.NewPort("tcp", dindPort)
	tcpPort := make(map[nat.Port]struct{})
	tcpPort[port] = struct{}{}
	volumeMounts := []string{
		sharedVolume + ":" + sharedVolume,
	}
	portBindings := nat.PortMap{
		port: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: dindPort}},
	}

	resp, err := r.hostClient.ContainerCreate(ctx,
		&container.Config{
			Image:        dindImageName,
			ExposedPorts: tcpPort,
		},
		&container.HostConfig{
			Binds:        volumeMounts,
			PortBindings: portBindings,
			RestartPolicy: container.RestartPolicy{
				Name:              "on-failure",
				MaximumRetryCount: 10,
			},
			Privileged: true,
		},
		nil,
		nil,
		dindContainerName)
	if err != nil {
		return err
	}

	err = r.hostClient.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return err
	}

	return nil
}
