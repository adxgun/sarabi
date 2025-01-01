package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	dockerclient "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"os"
	"os/exec"
	"sarabi/internal/bundler"
	"sarabi/internal/types"
	"sarabi/logger"
	"strings"
	"time"
)

type Docker interface {
	BuildImage(ctx context.Context, application *types.Deployment) (BuildImageResult, error)
	IsContainerRunning(ctx context.Context, container string) (bool, ContainerInfo, error)
	CreateNetwork(ctx context.Context, name string) error
	PullImage(ctx context.Context, name string) error
	StartContainerAndWait(ctx context.Context, params StartContainerParams) (*ContainerInfo, error)
	RestartContainer(ctx context.Context, name string) error
	StopAndRemoveContainer(ctx context.Context, param StopContainerParams) error
	CopyFileIntoContainer(ctx context.Context, containerName, src, dest string) error
	ExtractFiles(ctx context.Context, containerName, fileDir string) error
	ConnectContainer(ctx context.Context, containerName, networkName string) error
	ContainerExec(ctx context.Context, params ContainerExecParams) (io.Reader, error)
	CopyFromContainer(ctx context.Context, containerName, filePath string) (types.File, error)
	ContainerStatus(ctx context.Context, name string) (string, error)
	ContainerLogs(ctx context.Context, name string) (io.ReadCloser, error)
	ContainerEvents(ctx context.Context) (<-chan events.Message, <-chan error)
}

type dockerClient struct {
	hostClient client.APIClient
}

func NewClient() (Docker, error) {
	hostClient, err := client.NewClientWithOpts(client.FromEnv,
		client.WithAPIVersionNegotiation(), client.WithTimeout(10*time.Minute))
	if err != nil {
		return nil, err
	}

	p, err := hostClient.Ping(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to docker host")
	}

	logger.Info("docker client connected",
		zap.Any("properties", p))
	return &dockerClient{hostClient: hostClient}, nil
}

func (d *dockerClient) BuildImage(ctx context.Context, application *types.Deployment) (BuildImageResult, error) {
	buildCtx, err := bundler.CreateBuildContextFromTar(application.BinPath())
	if err != nil {
		return BuildImageResult{}, err
	}

	imageName := application.ImageName()
	response, err := d.hostClient.ImageBuild(ctx, &buildCtx, dockerclient.ImageBuildOptions{
		Tags:        []string{imageName},
		Remove:      true,
		ForceRemove: true,
	})
	if err != nil {
		return BuildImageResult{}, err
	}

	defer func() {
		_ = response.Body.Close()
	}()
	resp, err := readRemoteResponse(response.Body)
	if err != nil {
		return BuildImageResult{}, err
	}
	for _, next := range resp {
		if next.ErrorDetails != nil {
			return BuildImageResult{}, fmt.Errorf("code: %d, message: %s", next.ErrorDetails.Code, next.ErrorDetails.Message)
		}
		if next.Stream != "" && strings.Contains(next.Stream, "successfully built") {
			return BuildImageResult{Name: imageName}, nil
		}
	}

	return BuildImageResult{Name: imageName}, nil
}

func (d *dockerClient) IsContainerRunning(ctx context.Context, container string) (bool, ContainerInfo, error) {
	result, err := d.hostClient.ContainerInspect(ctx, container)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, ContainerInfo{}, nil
		}
		return false, ContainerInfo{}, err
	}

	if result.State.Running || result.State.Restarting {
		return true, ContainerInfo{ID: result.ID, Name: result.Name}, nil
	}

	return false, ContainerInfo{}, nil
}

func (d *dockerClient) CreateNetwork(ctx context.Context, name string) error {
	r, err := d.hostClient.NetworkInspect(ctx, name, network.InspectOptions{})
	if err == nil && r.ID != "" {
		return nil
	}

	_, err = d.hostClient.NetworkCreate(ctx, name, network.CreateOptions{
		Driver: "bridge",
	})
	if err != nil {
		return err
	}
	return nil
}

func (d *dockerClient) PullImage(ctx context.Context, name string) error {
	result, err := d.hostClient.ImagePull(ctx, name, image.PullOptions{})
	if err != nil {
		return err
	}

	defer result.Close()
	_, err = readRemoteResponse(result)
	if err != nil {
		return err
	}

	return nil
}

func (d *dockerClient) StartContainerAndWait(ctx context.Context, params StartContainerParams) (*ContainerInfo, error) {
	portSet := make(map[nat.Port]struct{})
	for _, ep := range params.ExposedPorts {
		portSet[ep] = struct{}{}
	}

	var containerNetwork *network.NetworkingConfig
	if params.Network != "" {
		containerNetwork = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				params.Network: {},
			},
		}
	}

	resp, err := d.hostClient.ContainerCreate(ctx,
		&container.Config{
			Env:          params.Environments,
			Image:        params.Image,
			ExposedPorts: portSet,
		},
		&container.HostConfig{
			Binds:        params.Volumes,
			PortBindings: params.PortBindings,
			NetworkMode:  network.NetworkBridge,
			RestartPolicy: container.RestartPolicy{
				Name:              "on-failure",
				MaximumRetryCount: 10,
			},
		},
		containerNetwork,
		nil,
		params.Container)
	if err != nil {
		return nil, err
	}

	if err := d.hostClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, err
	}

	isRunning, info, err := d.IsContainerRunning(ctx, resp.ID)
	for !isRunning {
		time.Sleep(200 * time.Millisecond)
		isRunning, info, err = d.IsContainerRunning(ctx, resp.ID)
		if err != nil {
			break
		}
	}
	return &info, nil
}

func (d *dockerClient) RestartContainer(ctx context.Context, name string) error {
	err := d.hostClient.ContainerRestart(ctx, name, container.StopOptions{})
	if err != nil {
		return err
	}

	d.wait(ctx, name)
	return nil
}

func (d *dockerClient) StopAndRemoveContainer(ctx context.Context, param StopContainerParams) error {
	err := d.hostClient.ContainerStop(ctx, param.ContainerName, container.StopOptions{})
	if err != nil {
		return err
	}

	return d.hostClient.ContainerRemove(ctx, param.ContainerName, container.RemoveOptions{
		RemoveVolumes: param.RemoveVolumes,
		Force:         true,
	})
}

func (d *dockerClient) CopyFileIntoContainer(ctx context.Context, containerName, src, fileDir string) error {
	gzFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer gzFile.Close()

	fileInfo, err := gzFile.Stat()
	if err != nil {
		return err
	}

	var buffer bytes.Buffer
	tarWriter := tar.NewWriter(&buffer)

	header := &tar.Header{
		Name: fileInfo.Name(),
		Mode: 0644,
		Size: fileInfo.Size(),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	if _, err := io.Copy(tarWriter, gzFile); err != nil {
		return err
	}
	if err := tarWriter.Close(); err != nil {
		return err
	}

	tarReader := bytes.NewReader(buffer.Bytes())
	err = d.hostClient.CopyToContainer(ctx, containerName, base64.StdEncoding.EncodeToString([]byte(fileDir)),
		tarReader, container.CopyToContainerOptions{
			AllowOverwriteDirWithFile: true,
		})
	if err != nil {
		return err
	}
	return nil
}

func (d *dockerClient) ExtractFiles(ctx context.Context, containerName, fileDir string) error {
	gzFilePath := base64.StdEncoding.EncodeToString([]byte(fileDir))
	cmd := []string{"tar", "-xzf", gzFilePath, "-C", fileDir}
	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err := d.hostClient.ContainerExecCreate(ctx, containerName, execConfig)
	if err != nil {
		return err
	}

	err = d.hostClient.ContainerExecStart(ctx, execID.ID, container.ExecStartOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (d *dockerClient) ConnectContainer(ctx context.Context, containerName, networkName string) error {
	err := d.hostClient.NetworkConnect(ctx, networkName, containerName, nil)
	if err != nil && strings.Contains(err.Error(), "already exists in network") {
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

// ContainerExec executes a command in the specified container and returns the output
// for some commands, the output is not immediately available(e.g a mysql dump command for a large db), so we wait for the command to finish
func (d *dockerClient) ContainerExec(ctx context.Context, params ContainerExecParams) (io.Reader, error) {
	execID, err := d.hostClient.ContainerExecCreate(ctx, params.ContainerName, container.ExecOptions{
		Env:          params.Envs,
		Cmd:          params.Cmd,
		AttachStderr: true,
		AttachStdout: true,
		Privileged:   true,
	})
	if err != nil {
		return nil, err
	}

	hr, err := d.hostClient.ContainerExecAttach(ctx, execID.ID, container.ExecAttachOptions{})
	if err != nil {
		return nil, err
	}

	for {
		inspect, err := d.hostClient.ContainerExecInspect(ctx, execID.ID)
		if err != nil {
			return nil, err
		}

		if !inspect.Running {
			if inspect.ExitCode == 0 {
				break
			} else {
				_, stdErr, err := ReadExecResponse(hr.Conn)
				if err != nil {
					return nil, err
				}
				return nil, fmt.Errorf("exec cmd error: %s", stdErr)
			}
		}
		time.Sleep(1 * time.Millisecond)
	}

	return hr.Conn, nil
}

// CopyFromContainer copies a file from a container to the host
func (d *dockerClient) CopyFromContainer(ctx context.Context, containerName, filePath string) (types.File, error) {
	tempFile := fmt.Sprintf("%s.sql", uuid.NewString())
	containerAndPath := fmt.Sprintf("%s:%s", containerName, filePath)
	cmd := exec.CommandContext(ctx, "docker", "cp", containerAndPath, tempFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		return types.File{}, errors.New(string(out))
	}

	fi, err := os.Open(tempFile)
	if err != nil {
		return types.File{}, err
	}

	stat, err := os.Stat(tempFile)
	if err != nil {
		return types.File{}, err
	}

	logger.Info("copy file from container successful",
		zap.String("container", containerName),
		zap.Any("size", stat.Size()),
		zap.String("name", stat.Name()))

	return types.File{
		Content: fi,
		Stat: types.FileStat{
			Size: stat.Size(),
			Name: stat.Name(),
			Mode: stat.Mode(),
		},
	}, nil
}

func (d *dockerClient) ContainerStatus(ctx context.Context, name string) (string, error) {
	result, err := d.hostClient.ContainerInspect(ctx, name)
	if err != nil {
		return "", err
	}

	return result.State.Status, nil
}

func (d *dockerClient) ContainerLogs(ctx context.Context, name string) (io.ReadCloser, error) {
	return d.hostClient.ContainerLogs(ctx, name, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       "all",
		Details:    false,
	})
}

func (d *dockerClient) ContainerEvents(ctx context.Context) (<-chan events.Message, <-chan error) {
	args := filters.NewArgs(filters.Arg("type", "container"))
	return d.hostClient.Events(ctx, events.ListOptions{
		Filters: args,
	})
}

func (d *dockerClient) wait(ctx context.Context, containerID string) {
	isRunning, _, err := d.IsContainerRunning(ctx, containerID)
	if err != nil {
		return
	}
	for !isRunning {
		time.Sleep(200 * time.Millisecond)
		isRunning, _, err = d.IsContainerRunning(ctx, containerID)
		if err != nil {
			break
		}
	}
}
