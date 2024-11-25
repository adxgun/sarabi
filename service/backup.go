package service

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/strslice"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	errors2 "github.com/pkg/errors"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"sarabi/database"
	"sarabi/integrations/docker"
	"sarabi/logger"
	"sarabi/storage"
	"sarabi/types"
	"strings"
	"time"
)

type (
	BackupService interface {
		Run(ctx context.Context) error
		CreateBackupSettings(ctx context.Context, applicationID uuid.UUID, environment string, duration time.Duration) error
	}

	backupService struct {
		dockerClient             docker.Docker
		applicationService       ApplicationService
		secretService            SecretService
		st                       storage.Storage
		backupSettingsRepository database.BackupSettingsRepository
		scheduler                gocron.Scheduler
		started                  bool
	}
)

func NewBackupService(dc docker.Docker, service ApplicationService,
	ss SecretService, st storage.Storage, backupSettings database.BackupSettingsRepository) (BackupService, error) {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}
	return &backupService{
		dockerClient:             dc,
		applicationService:       service,
		secretService:            ss,
		st:                       st,
		backupSettingsRepository: backupSettings,
		scheduler:                scheduler,
	}, nil
}

func (b backupService) Run(ctx context.Context) error {
	all, err := b.backupSettingsRepository.FindAll(ctx)
	if err != nil {
		return err
	}

	for _, bc := range all {
		if err := b.runScheduler(ctx, bc); err != nil {
			return err
		}
	}
	return nil
}

func (b backupService) run(ctx context.Context, settings *types.BackupSettings) error {
	application, err := b.applicationService.Get(ctx, settings.ApplicationID)
	if err != nil {
		return err
	}

	allAppVars, err := b.secretService.FindAll(ctx, settings.ApplicationID)
	if err != nil {
		return err
	}

	dbVars := lo.Filter(allAppVars, func(item *types.Secret, index int) bool {
		return types.InstanceType(item.InstanceType) == types.InstanceTypeDatabase &&
			item.Environment == settings.Environment
	})

	for _, se := range application.StorageEngines {
		switch se {
		case types.StorageEnginePostgres:
			err = b.postgresBackup(ctx, application, dbVars, settings.Environment)
			if err != nil {
				logger.Error("pg backup returned error",
					zap.Error(err))
			} else {
				logger.Info("backup completed",
					zap.String("application", application.Name),
					zap.String("environment", settings.Environment),
					zap.String("engine", string(se)),
					zap.String("ts", time.Now().String()))
			}
		default:
			return nil
		}
	}

	return nil
}

func (b backupService) runScheduler(ctx context.Context, bc *types.BackupSettings) error {
	job, err := b.scheduler.NewJob(
		gocron.DurationJob(bc.BackupInterval),
		gocron.NewTask(b.run, ctx, bc),
		gocron.WithSingletonMode(gocron.LimitModeWait))
	if err != nil {
		return err
	}

	logger.Info("backup job queued",
		zap.String("Name", job.Name()),
		zap.String("environment", bc.Environment))

	if !b.started {
		b.scheduler.Start()
	}
	b.started = true
	return nil
}

func (b backupService) postgresBackup(ctx context.Context, application *types.Application, vars []*types.Secret, environment string) error {
	logger.Info("starting postgres backup",
		zap.Any("application", application.Name),
		zap.String("env", environment))
	uName, err := FindSecret("POSTGRES_USER", vars)
	if err != nil {
		return err
	}
	password, err := FindSecret("POSTGRES_PASSWORD", vars)
	if err != nil {
		return err
	}
	dbName, err := FindSecret("POSTGRES_DB", vars)
	if err != nil {
		return err
	}

	resultPath := fmt.Sprintf("tmp/%s.sql", uuid.NewString())
	cmd := strslice.StrSlice{
		"pg_dump",
		"-U", uName.Value,
		"-d", dbName.Value,
		"-f", resultPath,
	}
	envs := []string{
		"PGPASSWORD=" + password.Value,
	}

	// /var/sarabi/data/backups/application_name-env/postgres_current_timestamp_formatted.sql
	location := fmt.Sprintf("%s/%s-%s/postgres-%s.sql", storage.BackupDir, application.Name, environment, time.Now().Format("2006_01_02_03_04pm"))
	r, err := b.dockerClient.ContainerExec(ctx, docker.ContainerExecParams{
		ContainerName: fmt.Sprintf("postgres-%s-%s", application.Name, environment),
		ResultPath:    resultPath,
		Cmd:           cmd,
		Envs:          envs,
	})
	if err != nil {
		return errors2.Wrap(err, "failed to execute pg_dump")
	}

	defer r.Close()
	return b.st.Save(ctx, location, r)
}

func (b backupService) CreateBackupSettings(ctx context.Context, applicationID uuid.UUID, environment string, duration time.Duration) error {
	settings, err := b.backupSettingsRepository.FindByApplicationID(ctx, applicationID)
	if err != nil {
		return errors2.Wrap(err, "failed to fetch backup settings")
	}

	exists := lo.Filter(settings, func(item *types.BackupSettings, index int) bool {
		return strings.ToLower(item.Environment) == strings.ToLower(environment)
	})
	if len(exists) > 0 {
		return nil
	}

	backupSettings := &types.BackupSettings{
		ID:             uuid.New(),
		ApplicationID:  applicationID,
		Environment:    environment,
		BackupInterval: duration,
		CreatedAt:      time.Now(),
	}
	err = b.backupSettingsRepository.Save(ctx, backupSettings)
	if err != nil {
		return err
	}

	return b.runScheduler(ctx, backupSettings)
}
