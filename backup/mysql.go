package backup

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/strslice"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"sarabi/integrations/docker"
	"sarabi/logger"
	"sarabi/storage"
	"sarabi/types"
	"time"
)

type (
	mysqlBackupExecutor struct {
		dockerClient docker.Docker
	}
)

func NewMysql(dc docker.Docker) Executor {
	return &mysqlBackupExecutor{dockerClient: dc}
}

func (m mysqlBackupExecutor) Execute(ctx context.Context, params ExecuteParams) (ExecuteResponse, error) {
	logger.Info("starting mysql backup",
		zap.String("application", params.Application.Name),
		zap.String("env", params.Environment))
	username, err := findVar("MYSQL_USER", params.DatabaseVars)
	if err != nil {
		return ExecuteResponse{}, err
	}

	password, err := findVar("MYSQL_PASSWORD", params.DatabaseVars)
	if err != nil {
		return ExecuteResponse{}, err
	}

	dbName, err := findVar("MYSQL_DATABASE", params.DatabaseVars)
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
		st, err = storage.NewS3Storage(*params.StorageCredential)
		if err != nil {
			return ExecuteResponse{}, errors.Wrap(err, "invalid object storage credential")
		}
		stType = storage.TypeS3
	}

	cmd := strslice.StrSlice{
		"mysqldump",
		"-u", username.Value,
		fmt.Sprintf("-p%s", password.Value),
		"-d", dbName.Value,
	}

	containerName := fmt.Sprintf("mysql-%s-%s", params.Application.Name, params.Environment)
	reader, err := m.dockerClient.ContainerExec(ctx, docker.ContainerExecParams{
		ContainerName: containerName,
		Cmd:           cmd,
	})
	if err != nil {
		return ExecuteResponse{}, errors.Wrap(err, "failed to execute mysqldump")
	}

	location := fmt.Sprintf("%s/%s-%s/mysql-%s.sql", storage.BackupDir, params.Application.Name, params.Environment, time.Now().Format("2006_01_02_03_04pm"))
	dmpFile := types.File{
		Content: reader,
	}

	if err := st.Save(ctx, location, dmpFile); err != nil {
		return ExecuteResponse{}, errors.Wrap(err, "failed to save file in storage")
	}

	return ExecuteResponse{
		Location:    location,
		StorageType: stType,
	}, nil
}
