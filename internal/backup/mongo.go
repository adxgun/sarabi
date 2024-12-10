package backup

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/strslice"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"sarabi/internal/integrations/docker"
	storage "sarabi/internal/storage"
	"sarabi/internal/types"
	"sarabi/logger"
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
	username, err := findVar("MONGO_INITDB_ROOT_USERNAME", params.DatabaseVars)
	if err != nil {
		return ExecuteResponse{}, err
	}
	password, err := findVar("MONGO_INITDB_ROOT_PASSWORD", params.DatabaseVars)
	if err != nil {
		return ExecuteResponse{}, err
	}

	var st storage.Storage
	var stType storage.Type
	if params.StorageCredential == nil {
		logger.Info("Object storage credential not configured, using File system storage for backup")
		st = storage.NewFileStorage()
		stType = storage.TypeFS
	} else {
		st, err = storage.NewObjectStorage(*params.StorageCredential)
		if err != nil {
			return ExecuteResponse{}, errors.Wrap(err, "invalid object storage credential")
		}
		stType = storage.TypeS3
	}

	resultPath := fmt.Sprintf("tmp/%s.tar", uuid.NewString())
	//  mongodump -u user -p password --out file.tar
	cmd := strslice.StrSlice{
		"mongodump",
		"-u", username.Value,
		"-p", password.Value,
		"--out",
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

	defer func() {
		_ = dmpFile.Content.Close()
	}()

	f := types.File{
		Content: dmpFile.Content,
		// -1 allows the object storage sdk(minio) detects the size
		Stat: types.FileStat{Size: -1, Name: dmpFile.Stat.Name},
	}

	if err := st.Save(ctx, location, f); err != nil {
		return ExecuteResponse{}, errors.Wrap(err, "failed to save file in storage")
	}

	return ExecuteResponse{
		Location:    location,
		StorageType: stType,
	}, nil
}
