package database

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sarabi/types"
)

type deploymentRepository struct {
	db *gorm.DB
}

func NewDeploymentRepository(db *gorm.DB) DeploymentRepository {
	return &deploymentRepository{db: db}
}

func (d *deploymentRepository) Save(ctx context.Context, deployment *types.Deployment) error {
	return d.db.WithContext(ctx).Save(deployment).Error
}

func (d *deploymentRepository) FindAll(ctx context.Context, applicationID uuid.UUID) ([]*types.Deployment, error) {
	values := make([]*types.Deployment, 0)
	err := d.db.WithContext(ctx).
		Preload("Application").
		Where("application_id = ?", applicationID).
		Find(&values).Error
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (d *deploymentRepository) FindByID(ctx context.Context, deploymentID uuid.UUID) (*types.Deployment, error) {
	dep := &types.Deployment{}
	err := d.db.WithContext(ctx).
		Preload("Application").
		Where("id = ?", deploymentID).
		First(dep).Error
	if err != nil {
		return nil, err
	}
	return dep, nil
}

func (d *deploymentRepository) UpdateDeploymentStatus(ctx context.Context, deploymentID uuid.UUID, newStatus string) error {
	return d.db.WithContext(ctx).
		Table("deployments").
		Where("id = ?", deploymentID).
		Update("status", newStatus).
		Error
}
