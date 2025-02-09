package logcollector

import (
	"context"
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"sarabi/internal/components"
	"sarabi/internal/integrations/docker"
	"sarabi/internal/integrations/loki"
	"sarabi/internal/service"
)

const (
	tmpConfigPath = "/etc/loki/config.yaml"
	imageName     = "grafana/loki:3.3.2"
)

type logCollector struct {
	dc          docker.Docker
	lc          loki.Client
	varsService service.SecretService
}

func New(dc docker.Docker, lc loki.Client, varsService service.SecretService) components.Builder {
	return &logCollector{
		dc:          dc,
		lc:          lc,
		varsService: varsService,
	}
}

func (l *logCollector) Name() string {
	return "sarabi-loki-log-collector"
}

func (l *logCollector) Run(ctx context.Context, deploymentID uuid.UUID) (*components.BuilderResult, error) {
	running, info, err := l.dc.IsContainerRunning(ctx, l.Name())
	if err != nil {
		return nil, err
	}

	if running {
		return &components.BuilderResult{ID: info.ID, Name: info.Name}, nil
	}

	if err := l.fixOwnership(ctx); err != nil {
		return nil, err
	}

	cfg := loki.DefaultConfig()
	// use filesystem as log store if S3 compatible storage is not configured
	storageCred, err := l.varsService.FindStorageCredentials(ctx, uuid.Nil)
	if err == nil && storageCred != nil {
		cfg.StorageConfig.AWS.S3 = storageCred.URI()
		cfg.StorageConfig.FileSystem = nil
		cfg.SchemaConfig.Configs[0].ObjectStore = "s3"
	} else {
		cfg.StorageConfig.AWS = nil
		cfg.SchemaConfig.Configs[0].ObjectStore = "filesystem"
		cfg.StorageConfig.FileSystem = &loki.FileSystemConfig{Directory: "/var/loki/chunks"}
	}

	cfgContent, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	if err := l.writeLokiInitConfig(cfgContent); err != nil {
		return nil, err
	}

	mounts := map[string]string{
		l.Name() + "-volume": "/var/loki",
	}

	cmd := []string{
		"-config.file=/etc/loki/config.yaml",
	}

	volumes := []string{
		fmt.Sprintf("/etc/loki/config.yaml:/etc/loki/config.yaml"),
	}
	tcpPort, _ := nat.NewPort("tcp", "3100")
	exposedPorts := []nat.Port{tcpPort}
	portBindings := nat.PortMap{
		tcpPort: []nat.PortBinding{{HostIP: "127.0.0.1", HostPort: "3100"}},
	}

	resp, err := l.dc.StartContainerAndWait(ctx, docker.StartContainerParams{
		Image:        imageName,
		Container:    l.Name(),
		Volumes:      volumes,
		Cmd:          cmd,
		ExposedPorts: exposedPorts,
		Mounts:       mounts,
		PortBindings: portBindings,
	})
	if err != nil {
		return nil, err
	}

	return &components.BuilderResult{ID: resp.ID, Name: info.Name}, nil
}

func (l *logCollector) Cleanup(ctx context.Context, result *components.BuilderResult) error {
	return nil
}

func (l *logCollector) writeLokiInitConfig(content []byte) error {
	if _, err := os.Stat(tmpConfigPath); err == nil {
		if err := os.Remove(tmpConfigPath); err != nil {
			return err
		}
	}

	dir := filepath.Dir(tmpConfigPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	fi, err := os.Create(tmpConfigPath)
	if err != nil {
		return err
	}

	_, err = io.WriteString(fi, string(content))
	if err != nil {
		return err
	}
	return nil
}

func (l *logCollector) fixOwnership(ctx context.Context) error {
	volumeName := l.Name() + "-volume"
	if err := l.dc.CreateNetwork(ctx, volumeName); err != nil {
		return err
	}
	param := docker.StartContainerParams{
		Image: "busybox:1.37.0",
		Cmd:   []string{"chown", "-R", "10001:10001", "/var/loki"},
		Mounts: map[string]string{
			volumeName: "/var/loki",
		},
		User: "0:0",
	}

	resp, err := l.dc.StartContainerAndWait(ctx, param)
	if err != nil {
		return err
	}

	return l.dc.StopAndRemoveContainer(ctx, docker.StopContainerParams{
		ContainerName: resp.Name,
	})
}
