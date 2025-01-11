package backup

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/strslice"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"os"
	"sarabi/internal/integrations/docker"
	storage "sarabi/internal/storage"
	"sarabi/internal/types"
	"sarabi/logger"
	"time"
)

type (
	postgresBackupExecutor struct {
		dockerClient docker.Docker
	}
)

func NewPostgres(dc docker.Docker) Executor {
	return &postgresBackupExecutor{dockerClient: dc}
}

func (p postgresBackupExecutor) Execute(ctx context.Context, params Params) (Result, error) {
	logger.Info("starting postgres backup",
		zap.String("application", params.Application.Name),
		zap.String("env", params.Environment))
	username, err := findVar("POSTGRES_USER", params.DatabaseVars)
	if err != nil {
		return Result{}, err
	}

	password, err := findVar("POSTGRES_PASSWORD", params.DatabaseVars)
	if err != nil {
		return Result{}, err
	}

	dbName, err := findVar("POSTGRES_DB", params.DatabaseVars)
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

	resultPath := fmt.Sprintf("/tmp/%s.sql", uuid.NewString())
	cmd := strslice.StrSlice{
		"pg_dump",
		"-U", username.Value,
		"-d", dbName.Value,
		"-f", resultPath,
	}
	envs := []string{
		"PGPASSWORD=" + password.Value,
	}

	location := fmt.Sprintf("%s/%s-%s/postgres-%s.sql", storage.BackupDir, params.Application.Name, params.Environment, time.Now().Format("2006_01_02_03_04pm"))
	containerName := fmt.Sprintf("postgres-%s-%s", params.Application.Name, params.Environment)
	_, err = p.dockerClient.ContainerExec(ctx, docker.ContainerExecParams{
		ContainerName: containerName,
		Cmd:           cmd,
		Envs:          envs,
	})
	if err != nil {
		return Result{}, errors.Wrap(err, "failed to execute pg_dump")
	}

	dmpFile, err := p.dockerClient.CopyFromContainer(ctx, containerName, resultPath)
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

func findVar(name string, in []*types.Secret) (*types.Secret, error) {
	for _, next := range in {
		if next.Name == name {
			return next, nil
		}
	}
	return nil, fmt.Errorf("var: %s was not found", name)
}
