package service

import (
	"context"
	"errors"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	errors2 "github.com/pkg/errors"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"sarabi/backup"
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
		CreateBackupSettings(ctx context.Context, applicationID uuid.UUID, environment string, runInterval time.Duration) error
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
	scheduler, err := gocron.NewScheduler()
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
	storageCred, _ := b.findStorageCredential(ctx, application)
	param := backup.ExecuteParams{
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
		default:
			return nil
		}
		result, err := bk.Execute(ctx, param)
		if err != nil {
			logger.Error("backup returned error",
				zap.Error(err),
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
			}
			if err := b.backupRepository.Save(ctx, newBackup); err != nil {
				logger.Error("failed to save backup", zap.Error(err))
			}
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

func (b backupService) CreateBackupSettings(ctx context.Context, applicationID uuid.UUID, environment string, runInterval time.Duration) error {
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
		BackupInterval: runInterval,
		CreatedAt:      time.Now(),
	}
	err = b.backupSettingsRepository.Save(ctx, backupSettings)
	if err != nil {
		return err
	}

	return b.runScheduler(ctx, backupSettings)
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
		st, err = storage.NewS3Storage(*cred)
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
	credentials, err := b.secretService.FindApplicationCredentials(ctx, application.ID, types.CredentialProviderS3)
	if err != nil {
		return nil, err
	}

	keyId, err := FindCredential("ACCESS_KEY", credentials)
	if err != nil {
		return nil, err
	}
	secretAccessKey, err := FindCredential("SECRET_KEY", credentials)
	if err != nil {
		return nil, err
	}
	endpoint, err := FindCredential("ENDPOINT", credentials)
	if err != nil {
		return nil, err
	}
	region, _ := FindCredential("REGION", credentials)
	regionStr := ""
	if region != nil {
		regionStr = region.Value
	}

	return &types.StorageCredentials{
		Endpoint:  endpoint.Value,
		KeyId:     keyId.Value,
		SecretKey: secretAccessKey.Value,
		Region:    regionStr,
	}, nil
}
