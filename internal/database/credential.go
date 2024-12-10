package database

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sarabi/internal/types"
)

type (
	serverConfigRepository struct {
		db *gorm.DB
	}
)

func NewServerConfigRepository(db *gorm.DB) ServerConfigRepository {
	return &serverConfigRepository{db: db}
}

func (c *serverConfigRepository) Save(ctx context.Context, cfg *types.ServerConfig) error {
	return c.db.WithContext(ctx).Save(cfg).Error
}

func (c *serverConfigRepository) FindByApplicationID(ctx context.Context, applicationID uuid.UUID) ([]*types.ServerConfig, error) {
	result := make([]*types.ServerConfig, 0)
	err := c.db.WithContext(ctx).
		Where("application_id = ?", applicationID).
		Find(&result).Error
	return result, err
}

func (c *serverConfigRepository) UpdateServerConfigValue(ctx context.Context, id uuid.UUID, newValue string) error {
	return c.db.WithContext(ctx).
		Table("server_configs").
		Where("id = ?", id).
		Update("value", newValue).
		Error
}

func (c *serverConfigRepository) FindByName(ctx context.Context, applicationID uuid.UUID, provider, name string) (*types.ServerConfig, error) {
	cred := &types.ServerConfig{}
	err := c.db.WithContext(ctx).
		Where("application_id = ? and name = ? and provider = ?", applicationID, name, provider).
		First(cred).Error
	return cred, err
}
