package docker

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
	"io"
	"sarabi/logger"
)

type RemoteResponse struct {
	Stream string `json:"stream"`
	Aux    struct {
		ID string `json:"ID"`
	}
	ErrorDetails *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errorDetail"`
}

type BuildImageResult struct {
	Name string
}

type ContainerInfo struct {
	ID   string
	Name string
}

type StartContainerParams struct {
	Image        string
	Container    string
	Network      string
	Volumes      []string
	Environments []string
	ExposedPorts []nat.Port
	PortBindings nat.PortMap
}

type ContainerExecParams struct {
	ContainerName string
	ResultPath    string
	Cmd           strslice.StrSlice
	Envs          []string
}

func readRemoteResponse(body io.Reader) ([]RemoteResponse, error) {
	resp := make([]RemoteResponse, 0)
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		nextLine := scanner.Text()
		r := &RemoteResponse{}
		if err := json.Unmarshal([]byte(nextLine), r); err != nil {
			logger.Warn("error parsing docker build response",
				zap.String("line", nextLine))
		}
		resp = append(resp, *r)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return resp, nil
}

func readExecResponse(reader io.Reader) (string, string, error) {
	var stdOut, stdErr bytes.Buffer
	_, err := stdcopy.StdCopy(&stdOut, &stdErr, reader)
	if err != nil {
		return "", "", err
	}
	return stdOut.String(), stdErr.String(), nil
}
