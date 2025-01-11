package backup

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"os"
	"sarabi/internal/integrations/docker"
	storage "sarabi/internal/storage"
	"sarabi/logger"
	"time"
)

type (
	redisBackupExecutor struct {
		dockerClient docker.Docker
	}
)

func NewRedis(dc docker.Docker) Executor {
	return &redisBackupExecutor{dockerClient: dc}
}

func (m redisBackupExecutor) Execute(ctx context.Context, params Params) (Result, error) {
	logger.Info("starting redis backup",
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
		st, err = storage.NewObjectStorage(*params.StorageCredential)
		if err != nil {
			return Result{}, errors.Wrap(err, "invalid object storage credential")
		}
		stType = storage.TypeS3
	}

	resultPath := "/data/dump.rdb"
	location := fmt.Sprintf("%s/%s-%s/redis-%s.rdb", storage.BackupDir, params.Application.Name, params.Environment, time.Now().Format("2006_01_02_03_04pm"))
	containerName := fmt.Sprintf("redis-%s-%s", params.Application.Name, params.Environment)
	dmpFile, err := m.dockerClient.CopyFromContainer(ctx, containerName, resultPath)
	if err != nil {
		return Result{}, errors.Wrap(err, "failed to copy dump file")
	}

	defer func() {
		_ = dmpFile.Content.Close()
		_ = os.Remove(dmpFile.Stat.Name)
	}()

	if err := st.Save(ctx, location, dmpFile); err != nil {
		return Result{}, errors.Wrap(err, "failed to save file in storage")
	}

	return Result{
		Location:    location,
		StorageType: stType,
		Size:        dmpFile.Stat.Size,
	}, nil
}
