package database

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sarabi/internal/types"
)

type (
	credentialRepository struct {
		db *gorm.DB
	}
)

func NewCredentialRepository(db *gorm.DB) CredentialRepository {
	return &credentialRepository{db: db}
}

func (c *credentialRepository) Save(ctx context.Context, cred *types.Credential) error {
	return c.db.WithContext(ctx).Save(cred).Error
}

func (c *credentialRepository) FindByApplicationID(ctx context.Context, applicationID uuid.UUID) ([]*types.Credential, error) {
	result := make([]*types.Credential, 0)
	err := c.db.WithContext(ctx).
		Where("application_id = ?", applicationID).
		Find(&result).Error
	return result, err
}

func (c *credentialRepository) UpdateCredentialValue(ctx context.Context, id uuid.UUID, newValue string) error {
	return c.db.WithContext(ctx).
		Table("credentials").
		Where("id = ?", id).
		Update("value", newValue).
		Error
}

func (c *credentialRepository) FindByName(ctx context.Context, applicationID uuid.UUID, provider, name string) (*types.Credential, error) {
	cred := &types.Credential{}
	err := c.db.WithContext(ctx).
		Where("application_id = ? and name = ? and provider = ?", applicationID, name, provider).
		First(cred).Error
	return cred, err
}
