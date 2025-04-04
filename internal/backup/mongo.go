package backup

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/strslice"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"os"
	"sarabi/internal/bundler"
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

func (m mongoBackupExecutor) Execute(ctx context.Context, params Params) (Result, error) {
	logger.Info("starting mongo backup",
		zap.String("application", params.Application.Name),
		zap.String("env", params.Environment))
	username, err := findVar("MONGO_INITDB_ROOT_USERNAME", params.DatabaseVars)
	if err != nil {
		return Result{}, err
	}
	password, err := findVar("MONGO_INITDB_ROOT_PASSWORD", params.DatabaseVars)
	if err != nil {
		return Result{}, err
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
			return Result{}, errors.Wrap(err, "invalid object storage credential")
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
		return Result{}, errors.Wrap(err, "failed to execute mongodump")
	}

	dmpFile, err := m.dockerClient.CopyFromContainer(ctx, containerName, resultPath)
	if err != nil {
		return Result{}, errors.Wrap(err, "failed to copy dump file")
	}

	defer func() {
		_ = dmpFile.Content.Close()
		_ = os.RemoveAll(dmpFile.Stat.Name)
		_ = os.RemoveAll(fmt.Sprintf("%s.tar.gz", dmpFile.Stat.Name))
	}()

	fi, err := bundler.GzipToReader(dmpFile.Stat.Name)
	if err != nil {
		return Result{}, errors.Wrap(err, "failed to gzip file")
	}

	stat, err := fi.Stat()
	if err != nil {
		return Result{}, errors.Wrap(err, "failed to get file stat")
	}

	if err := st.Save(ctx, location, types.File{
		Content: fi,
		Stat: types.FileStat{
			Name: stat.Name(),
			Size: stat.Size(),
		},
	}); err != nil {
		return Result{}, errors.Wrap(err, "failed to save file in storage")
	}

	return Result{
		Location:    location,
		StorageType: stType,
		Size:        stat.Size(),
	}, nil
}
