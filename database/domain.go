package database

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sarabi/types"
)

type (
	domainRepository struct {
		db *gorm.DB
	}
)

func NewDomainRepository(db *gorm.DB) DomainRepository {
	return &domainRepository{db: db}
}

func (d *domainRepository) Save(ctx context.Context, domain *types.Domain) error {
	return d.db.
		WithContext(ctx).
		Save(domain).
		Error
}

func (d *domainRepository) FindByID(ctx context.Context, id uuid.UUID) (*types.Domain, error) {
	domain := &types.Domain{}
	err := d.db.
		WithContext(ctx).
		Where("id = ?", id).
		First(domain).Error
	if err != nil {
		return nil, err
	}
	return domain, nil
}

func (d *domainRepository) Find(ctx context.Context, name string) (*types.Domain, error) {
	domain := &types.Domain{}
	err := d.db.
		WithContext(ctx).
		Where("name = ?", name).
		First(domain).Error
	if err != nil {
		return nil, err
	}
	return domain, nil
}

func (d *domainRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return d.db.
		WithContext(ctx).
		Where("id = ?", id).
		Delete(&types.Domain{}).Error
}

func (d *domainRepository) FindByApplicationID(ctx context.Context, applicationID uuid.UUID) ([]*types.Domain, error) {
	result := make([]*types.Domain, 0)
	err := d.db.
		WithContext(ctx).
		Where("application_id = ?", applicationID).
		Find(&result).Error
	return result, err
}
