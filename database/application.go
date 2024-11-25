package database

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sarabi/types"
)

type applicationRepository struct {
	db *gorm.DB
}

func NewApplicationRepository(db *gorm.DB) ApplicationRepository {
	return &applicationRepository{db: db}
}

func (a *applicationRepository) Save(ctx context.Context, app *types.Application) error {
	return a.db.WithContext(ctx).Save(app).Error
}

func (a *applicationRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Application, error) {
	value := &types.Application{}
	err := a.db.
		WithContext(ctx).
		Where("id = ?", id).First(value).Error
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (a *applicationRepository) FindByName(ctx context.Context, name string) (*types.Application, error) {
	value := &types.Application{}
	err := a.db.
		WithContext(ctx).
		Where("name = ?", name).
		First(value).Error
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (a *applicationRepository) FindAll(ctx context.Context) ([]*types.Application, error) {
	resp := make([]*types.Application, 0)
	err := a.db.WithContext(ctx).Where("deleted_at = NULL").Find(&resp).Error
	return resp, err
}
