package database

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sarabi/internal/types"
)

type secretRepository struct {
	db *gorm.DB
}

func NewSecretRepository(db *gorm.DB) SecretRepository {
	return &secretRepository{db: db}
}

func (s *secretRepository) Save(ctx context.Context, secret *types.Secret) error {
	return s.db.WithContext(ctx).Save(secret).Error
}

func (s *secretRepository) FindAll(ctx context.Context, applicationID uuid.UUID) ([]*types.Secret, error) {
	values := make([]*types.Secret, 0)
	err := s.db.
		WithContext(ctx).
		Where("application_id = ?", applicationID).
		Find(&values).Error
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (s *secretRepository) FindBy(ctx context.Context, applicationID uuid.UUID, name, env, instanceType string) (*types.Secret, error) {
	value := &types.Secret{}
	err := s.db.
		WithContext(ctx).
		Where("application_id = ? AND name = ? AND environment = ? AND instance_type = ?", applicationID, name, env, instanceType).
		First(value).Error
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (s *secretRepository) UpdateValue(ctx context.Context, id uuid.UUID, newValue string) error {
	return s.db.WithContext(ctx).
		Table("secrets").
		Where("id = ?", id).
		Update("value", newValue).
		Error
}
