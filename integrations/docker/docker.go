package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	dockerclient "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
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
	"runtime"
	"sarabi/bundler"
	"sarabi/logger"
	"sarabi/types"
	"strings"
	"time"
)

type Docker interface {
	RunDind(ctx context.Context) error
	BuildImage(ctx context.Context, application *types.Deployment) (BuildImageResult, error)
	IsContainerRunning(ctx context.Context, container string) (bool, ContainerInfo, error)
	CreateNetwork(ctx context.Context, name string) error
	PullImage(ctx context.Context, name string) error
	StartContainerAndWait(ctx context.Context, params StartContainerParams) (*ContainerInfo, error)
	RestartContainer(ctx context.Context, name string) error
	StopAndRemoveContainer(ctx context.Context, containerName string) error
	CopyFileIntoContainer(ctx context.Context, containerName, src, dest string) error
	ExtractFiles(ctx context.Context, containerName, fileDir string) error
	ConnectContainer(ctx context.Context, containerName, networkName string) error
	ContainerExec(ctx context.Context, params ContainerExecParams) (io.Reader, error)
	CopyFromContainer(ctx context.Context, containerName, filePath string) (types.File, error)
}

type dockerClient struct {
	hostClient client.APIClient
	dindClient client.APIClient
	dindRunner Runner
}

func NewDockerClient() (Docker, error) {
	c, err := client.NewClientWithOpts(client.FromEnv,
		client.WithAPIVersionNegotiation(), client.WithTimeout(10*time.Minute))
	if err != nil {
		return nil, err
	}

	dindHost := fmt.Sprintf("tcp://localhost:%s", dindPort)
	dindClient, err := client.NewClientWithOpts(client.WithHost(dindHost), client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, errors.Wrap(err, "failed to create dind client")
	}
	return &dockerClient{hostClient: c, dindClient: dindClient, dindRunner: newRunner(c)}, nil
}

func (d *dockerClient) RunDind(ctx context.Context) error {
	return d.dindRunner.Run(ctx)
}

func (d *dockerClient) BuildImage(ctx context.Context, application *types.Deployment) (BuildImageResult, error) {
	buildCtx, err := bundler.CreateBuildContextFromTar(application.BinPath())
	if err != nil {
		return BuildImageResult{}, err
	}

	imageName := application.ImageName()
	var dockerCli = d.hostClient
	if runtime.GOOS == "linux" {
		dockerCli = d.dindClient
	}

	response, err := dockerCli.ImageBuild(ctx, &buildCtx, dockerclient.ImageBuildOptions{
		Tags:        []string{imageName},
		Remove:      true,
		ForceRemove: true,
	})
	if err != nil {
		return BuildImageResult{}, err
	}

	defer response.Body.Close()
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

	if runtime.GOOS == "linux" {
		saved, err := d.dindClient.ImageSave(ctx, []string{imageName})
		defer saved.Close()

		err = bundler.WriteToPath(saved, fmt.Sprintf("%s/%s.tar", sharedVolume, imageName))
		if err != nil {
			return BuildImageResult{}, err
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

	return false, ContainerInfo{}, errors.New("container is not running: " + string(result.State.Error))
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

	if runtime.GOOS == "linux" {
		tarPath := sharedVolume + "/" + params.Image + ".tar"
		fi, err := os.Open(tarPath)
		if err != nil {
			return nil, err
		}

		r, err := d.hostClient.ImageLoad(ctx, fi, false)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load image")
		}
		defer r.Body.Close()
	}

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

func (d *dockerClient) StopAndRemoveContainer(ctx context.Context, containerName string) error {
	err := d.hostClient.ContainerStop(ctx, containerName, container.StopOptions{})
	if err != nil {
		return err
	}

	return d.hostClient.ContainerRemove(ctx, containerName, container.RemoveOptions{
		RemoveVolumes: true,
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

	execResponse, err := d.hostClient.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return nil, err
	}

	if execResponse.ExitCode == 0 {
		return hr.Conn, nil
	}

	_, stdErr, err := ReadExecResponse(hr.Conn)
	if err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("exec cmd error: %s", stdErr)
}

// CopyFromContainer copy file from the specified container from the specified file path. Sometimes, it takes a few seconds for the file
// to be available e.g after a database dump, so we included a retry to wait for the file to be available
func (d *dockerClient) CopyFromContainer(ctx context.Context, containerName, filePath string) (types.File, error) {
	var (
		retries = 10
		delay   = 100 * time.Millisecond
	)

	for i := 0; i < retries; i++ {
		_, err := d.hostClient.ContainerStatPath(ctx, containerName, filePath)
		if err == nil {
			break
		}

		time.Sleep(delay)
		delay *= 2
	}

	ff := fmt.Sprintf("%s.sql", uuid.NewString())
	containerAndPath := fmt.Sprintf("%s:%s", containerName, filePath)
	cmd := exec.Command("docker", "cp", containerAndPath, ff)
	if err := cmd.Run(); err != nil {
		return types.File{}, err
	}

	fi, err := os.Open(ff)
	if err != nil {
		return types.File{}, err
	}

	stat, err := os.Stat(ff)
	if err != nil {
		return types.File{}, err
	}

	logger.Info("copy cmd",
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
