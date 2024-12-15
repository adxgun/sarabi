package database

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sarabi/internal/types"
)

type (
	backupSettingsRepository struct {
		db *gorm.DB
	}

	backupRepository struct {
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

func NewBackupRepository(db *gorm.DB) BackupRepository {
	return &backupRepository{db: db}
}

func (b backupRepository) Save(ctx context.Context, bc *types.Backup) error {
	return b.db.WithContext(ctx).Save(bc).Error
}

func (b backupRepository) FindByApplicationID(ctx context.Context, applicationID uuid.UUID) ([]*types.Backup, error) {
	result := make([]*types.Backup, 0)
	err := b.db.WithContext(ctx).Where("application_id = ?", applicationID).Find(&result).Error
	return result, err
}

func (b backupRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Backup, error) {
	bk := &types.Backup{}
	err := b.db.WithContext(ctx).Preload("Application").Where("id = ?", id).First(bk).Error
	return bk, err
}
