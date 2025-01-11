package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	errors2 "github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"sarabi/internal/backup"
	"sarabi/internal/database"
	"sarabi/internal/integrations/docker"
	"sarabi/internal/storage"
	"sarabi/internal/types"
	"sarabi/logger"
	"strings"
	"syscall"
	"time"
)

type (
	BackupService interface {
		Run(ctx context.Context) error
		CreateBackupSettings(ctx context.Context, applicationID uuid.UUID, environment string, cronExpression string, updateRunner bool) error
		Download(ctx context.Context, backupID uuid.UUID) (*types.File, error)
		ListBackups(ctx context.Context, applicationID uuid.UUID) ([]*types.Backup, error)
	}

	backupService struct {
		dockerClient             docker.Docker
		applicationService       ApplicationService
		secretService            SecretService
		backupSettingsRepository database.BackupSettingsRepository
		backupRepository         database.BackupRepository
		scheduler                gocron.Scheduler
		started                  bool
	}
)

func NewBackupService(dc docker.Docker, service ApplicationService,
	ss SecretService, backupSettings database.BackupSettingsRepository, repository database.BackupRepository) (BackupService, error) {
	scheduler, err := gocron.NewScheduler(
		gocron.WithLimitConcurrentJobs(10, gocron.LimitModeWait))
	if err != nil {
		return nil, err
	}
	return &backupService{
		dockerClient:             dc,
		applicationService:       service,
		secretService:            ss,
		backupSettingsRepository: backupSettings,
		scheduler:                scheduler,
		backupRepository:         repository,
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
	storageCred, err := b.findStorageCredential(ctx, application)
	param := backup.Params{
		Environment:       settings.Environment,
		DatabaseVars:      dbVars,
		StorageCredential: storageCred,
		Application:       application,
	}

	for _, se := range application.StorageEngines {
		var bk backup.Executor
		switch se {
		case types.StorageEnginePostgres:
			bk = backup.NewPostgres(b.dockerClient)
		case types.StorageEngineMysql:
			bk = backup.NewMysql(b.dockerClient)
		case types.StorageEngineMongo:
			bk = backup.NewMongo(b.dockerClient)
		case types.StorageEngineRedis:
			bk = backup.NewRedis(b.dockerClient)
		default:
			return nil
		}
		result, err := bk.Execute(ctx, param)
		if err != nil {
			logger.Error("backup returned error",
				zap.Error(err),
				zap.String("application", application.Name),
				zap.Any("storage_engine", se))
		} else {
			logger.Info("backup completed",
				zap.String("application", application.Name),
				zap.String("environment", settings.Environment),
				zap.String("engine", string(se)),
				zap.String("ts", time.Now().String()))

			newBackup := &types.Backup{
				ID:            uuid.New(),
				ApplicationID: application.ID,
				Environment:   settings.Environment,
				CreatedAt:     time.Now(),
				StorageEngine: se,
				Location:      result.Location,
				StorageType:   string(result.StorageType),
				Size:          result.Size,
			}
			if err := b.backupRepository.Save(ctx, newBackup); err != nil {
				logger.Error("failed to save backup", zap.Error(err))
			}
		}
	}

	return nil
}

func (b backupService) runBG(ctx context.Context, settings *types.BackupSettings) error {
	go func() {
		if err := b.run(ctx, settings); err != nil {
			logger.Info("run failed", zap.Error(err))
		}
	}()
	return nil
}

func (b backupService) runScheduler(ctx context.Context, bc *types.BackupSettings) error {
	job, err := b.scheduler.NewJob(
		gocron.CronJob(bc.CronExpression, false),
		gocron.NewTask(b.runBG, ctx, bc),
		gocron.WithIdentifier(bc.ID))
	if err != nil {
		return err
	}

	logger.Info("backup job queued",
		zap.String("Name", job.Name()),
		zap.String("expression", bc.CronExpression),
		zap.String("environment", bc.Environment))
	b.scheduler.Start()
	return nil
}

func (b backupService) CreateBackupSettings(
	ctx context.Context,
	applicationID uuid.UUID,
	environment string,
	cronExpression string,
	updateRunner bool) error {
	if err := b.parseExpression(cronExpression); err != nil {
		return err
	}

	settings, err := b.backupSettingsRepository.FindByApplicationID(ctx, applicationID)
	if err != nil {
		return errors2.Wrap(err, "failed to fetch backup settings")
	}

	exists := lo.Filter(settings, func(item *types.BackupSettings, index int) bool {
		return strings.ToLower(item.Environment) == strings.ToLower(environment)
	})
	if len(exists) > 0 {
		if updateRunner {
			return b.updateBackupSettings(ctx, exists[0], cronExpression)
		}
		return nil
	}

	backupSettings := &types.BackupSettings{
		ID:             uuid.New(),
		ApplicationID:  applicationID,
		Environment:    environment,
		CronExpression: cronExpression,
		CreatedAt:      time.Now(),
	}
	err = b.backupSettingsRepository.Save(ctx, backupSettings)
	if err != nil {
		return err
	}

	return b.runScheduler(ctx, backupSettings)
}

func (b backupService) updateBackupSettings(ctx context.Context, settings *types.BackupSettings, expression string) error {
	if err := b.backupSettingsRepository.UpdateExpression(ctx, settings.ID, expression); err != nil {
		return err
	}

	ctx, _ = signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	err := b.scheduler.RemoveJob(settings.ID)
	if errors.Is(err, gocron.ErrJobNotFound) {
		settings.CronExpression = expression
		return b.runScheduler(ctx, settings)
	}

	if err != nil {
		return err
	}

	settings.CronExpression = expression
	return b.runScheduler(ctx, settings)
}

func (b backupService) Download(ctx context.Context, backupID uuid.UUID) (*types.File, error) {
	bk, err := b.backupRepository.FindByID(ctx, backupID)
	if err != nil {
		return nil, err
	}

	cred, crederr := b.findStorageCredential(ctx, bk.Application)
	var st storage.Storage
	switch storage.Type(bk.StorageType) {
	case storage.TypeFS:
		st = storage.NewFileStorage()
	case storage.TypeS3:
		if cred == nil {
			return nil, errors2.Wrap(crederr, "failed to find object storage credential")
		}
		st, err = storage.NewObjectStorage(*cred)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unknown storage type")
	}

	return st.Get(ctx, bk.Location)
}

func (b backupService) ListBackups(ctx context.Context, applicationID uuid.UUID) ([]*types.Backup, error) {
	return b.backupRepository.FindByApplicationID(ctx, applicationID)
}

func (b backupService) findStorageCredential(ctx context.Context, application *types.Application) (*types.StorageCredentials, error) {
	credentials, err := b.secretService.FindApplicationServerConfigs(ctx, application.ID)
	if err != nil {
		return nil, err
	}

	objectStorageConfig := lo.Filter(credentials, func(item *types.ServerConfig, index int) bool {
		return item.Name == types.ServerConfigObjectStorage
	})
	if len(objectStorageConfig) == 0 {
		return nil, errors.New("no object storage configured")
	}

	value := objectStorageConfig[0]
	cred := &types.StorageCredentials{}
	if err := json.Unmarshal([]byte(value.Value), cred); err != nil {
		return nil, err
	}
	return cred, nil
}

func (b backupService) parseExpression(cronExpression string) error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(cronExpression)
	if err != nil {

		return fmt.Errorf("invalid cron expression: %w", err)
	}

	return nil
}
