package database

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sarabi/internal/types"
)

type networkAccessRepository struct {
	db *gorm.DB
}

func NewNetworkAccessRepository(db *gorm.DB) NetworkAccessRepository {
	return &networkAccessRepository{db: db}
}

func (n networkAccessRepository) Save(ctx context.Context, na *types.NetworkAccess) error {
	return n.db.
		WithContext(ctx).
		Save(na).Error
}

func (n networkAccessRepository) FindByApplicationID(ctx context.Context, applicationID uuid.UUID) ([]*types.NetworkAccess, error) {
	result := make([]*types.NetworkAccess, 0)
	err := n.db.WithContext(ctx).Where("application_id = ?", applicationID).Find(&result).Error
	return nil, err
}

func (n networkAccessRepository) FindByIP(ctx context.Context, ip string) ([]*types.NetworkAccess, error) {
	result := make([]*types.NetworkAccess, 0)
	err := n.db.WithContext(ctx).Where("ip = ?", ip).Find(&result).Error
	return nil, err
}

func (n networkAccessRepository) Remove(ctx context.Context, naID uuid.UUID) error {
	return n.db.
		WithContext(ctx).
		Where("id = ?", naID).
		Delete(&types.NetworkAccess{}).Error
}
