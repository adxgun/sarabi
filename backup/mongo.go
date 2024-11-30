package backup

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/strslice"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"sarabi/integrations/docker"
	"sarabi/logger"
	"sarabi/storage"
	"time"
)

type (
	mongoBackupExecutor struct {
		dockerClient docker.Docker
	}
)

func NewMongo(dc docker.Docker) Executor {
	return &mongoBackupExecutor{dockerClient: dc}
}

func (m mongoBackupExecutor) Execute(ctx context.Context, params ExecuteParams) (ExecuteResponse, error) {
	logger.Info("starting mongo backup",
		zap.String("application", params.Application.Name),
		zap.String("env", params.Environment))
	var st storage.Storage
	var stType storage.Type
	var err error
	if params.StorageCredential == nil {
		logger.Info("Object storage credential not configured, using File system storage for backup")
		st = storage.NewFileStorage()
		stType = storage.TypeFS
	} else {
		st, err = storage.NewS3Storage(*params.StorageCredential)
		if err != nil {
			return ExecuteResponse{}, errors.Wrap(err, "invalid object storage credential")
		}
		stType = storage.TypeS3
	}

	resultPath := fmt.Sprintf("tmp/%s.tar", uuid.NewString())
	cmd := strslice.StrSlice{
		"mongodump",
		"-out",
		resultPath,
	}

	location := fmt.Sprintf("%s/%s-%s/mongo-%s.tar", storage.BackupDir, params.Application.Name, params.Environment, time.Now().Format("2006_01_02_03_04pm"))
	containerName := fmt.Sprintf("mongo-%s-%s", params.Application.Name, params.Environment)
	_, err = m.dockerClient.ContainerExec(ctx, docker.ContainerExecParams{
		ContainerName: containerName,
		Cmd:           cmd,
	})
	if err != nil {
		return ExecuteResponse{}, errors.Wrap(err, "failed to execute mongodump")
	}

	dmpFile, err := m.dockerClient.CopyFromContainer(ctx, containerName, resultPath)
	if err != nil {
		return ExecuteResponse{}, errors.Wrap(err, "failed to copy dump file")
	}

	if err := st.Save(ctx, location, dmpFile); err != nil {
		return ExecuteResponse{}, errors.Wrap(err, "failed to save file in storage")
	}

	return ExecuteResponse{
		Location:    location,
		StorageType: stType,
	}, nil
}
