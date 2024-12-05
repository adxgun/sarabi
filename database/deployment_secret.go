package database

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sarabi/types"
)

type deploymentSecretRepository struct {
	db *gorm.DB
}

func NewDeploymentSecretRepository(db *gorm.DB) DeploymentSecretRepository {
	return &deploymentSecretRepository{db: db}
}

func (d *deploymentSecretRepository) SaveAll(ctx context.Context, applicationSecrets []*types.DeploymentSecret) error {
	return d.db.WithContext(ctx).Save(applicationSecrets).Error
}

func (d *deploymentSecretRepository) FindAll(ctx context.Context, deploymentID uuid.UUID) ([]*types.DeploymentSecret, error) {
	values := make([]*types.DeploymentSecret, 0)
	err := d.db.WithContext(ctx).
		Preload("Deployment").
		Preload("Secret").
		Where("deployment_id = ?", deploymentID).
		Find(&values).Error

	if err != nil {
		return nil, err
	}

	return values, nil
}

func (d *deploymentSecretRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return d.db.
		WithContext(ctx).
		Where("id = ?", id).
		Delete(&types.DeploymentSecret{}).Error
}
