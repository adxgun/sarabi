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
	"sarabi/logger"
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
		st, err = storage.NewObjectStorage(*params.StorageCredential)
		if err != nil {
			return ExecuteResponse{}, errors.Wrap(err, "invalid object storage credential")
		}
		stType = storage.TypeS3
	}

	// sh -c 'mysqldump -u sample-go-stage-user -p"password" mysql-sample-go-stage > /tmp/uuid.sql'
	resultPath := fmt.Sprintf("/tmp/%s.sql", uuid.NewString())
	cmdStr := fmt.Sprintf(`mysqldump -u %s -p"%s" %s > %s`,
		username.Value, password.Value, dbName.Value, resultPath)
	cmd := strslice.StrSlice{
		"sh",
		"-c",
		cmdStr,
	}
	envs := []string{
		"MYSQL_PWD=" + password.Value,
	}
	containerName := fmt.Sprintf("mysql-%s-%s", params.Application.Name, params.Environment)
	_, err = m.dockerClient.ContainerExec(ctx, docker.ContainerExecParams{
		ContainerName: containerName,
		Cmd:           cmd,
		Envs:          envs,
	})
	if err != nil {
		return ExecuteResponse{}, errors.Wrap(err, "failed to execute mysqldump")
	}

	location := fmt.Sprintf("%s/%s-%s/mysql-%s.sql", storage.BackupDir, params.Application.Name, params.Environment, time.Now().Format("2006_01_02_03_04pm"))
	dmpFile, err := m.dockerClient.CopyFromContainer(ctx, containerName, resultPath)
	if err != nil {
		return ExecuteResponse{}, err
	}

	defer func() {
		_ = dmpFile.Content.Close()
	}()

	if err := st.Save(ctx, location, dmpFile); err != nil {
		return ExecuteResponse{}, errors.Wrap(err, "failed to save file in storage")
	}

	return ExecuteResponse{
		Location:    location,
		StorageType: stType,
	}, nil
}
