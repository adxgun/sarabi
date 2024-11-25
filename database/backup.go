package database

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sarabi/types"
)

type (
	backupSettingsRepository struct {
		db *gorm.DB
	}
)

func NewBackupSettingsRepository(db *gorm.DB) BackupSettingsRepository {
	return &backupSettingsRepository{db: db}
}

func (b backupSettingsRepository) Save(ctx context.Context, settings *types.BackupSettings) error {
	return b.db.WithContext(ctx).Save(settings).Error
}

func (b backupSettingsRepository) FindAll(ctx context.Context) ([]*types.BackupSettings, error) {
	result := make([]*types.BackupSettings, 0)
	err := b.db.WithContext(ctx).Find(&result).Error
	return result, err
}

func (b backupSettingsRepository) FindByApplicationID(ctx context.Context, applicationID uuid.UUID) ([]*types.BackupSettings, error) {
	result := make([]*types.BackupSettings, 0)
	err := b.db.WithContext(ctx).Where("application_id = ?", applicationID).Find(&result).Error
	return result, err
}
