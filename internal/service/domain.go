package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"sarabi/internal/database"
	"sarabi/internal/types"
)

type (
	DomainService interface {
		AddDomain(ctx context.Context, applicationID uuid.UUID, params types.AddDomainParams) (*types.Domain, error)
		RemoveDomain(ctx context.Context, applicationID uuid.UUID, name string) (*types.Domain, error)
		FindByApplicationID(ctx context.Context, applicationID uuid.UUID) ([]*types.Domain, error)
		FindForEnvironmentAndInstanceType(ctx context.Context, applicationID uuid.UUID, environment string, ist types.InstanceType) ([]*types.Domain, error)
	}

	domainService struct {
		domainRepository database.DomainRepository
	}
)

func NewDomainService(domainRepo database.DomainRepository) DomainService {
	return &domainService{
		domainRepository: domainRepo,
	}
}

func (d *domainService) AddDomain(ctx context.Context, applicationID uuid.UUID, params types.AddDomainParams) (*types.Domain, error) {
	domain, err := d.domainRepository.Find(ctx, params.Name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if err == nil && domain.ID != uuid.Nil {
		return nil, fmt.Errorf("this domain already exists in envronment: %s", domain.Environment)
	}

	newDomain := &types.Domain{
		ID:            uuid.New(),
		ApplicationID: applicationID,
		Name:          params.Name,
		Environment:   params.Environment,
		InstanceType:  params.InstanceType,
		Status:        "CREATED",
	}
	err = d.domainRepository.Save(ctx, newDomain)
	if err != nil {
		return nil, err
	}

	return newDomain, nil
}

func (d *domainService) RemoveDomain(ctx context.Context, applicationID uuid.UUID, name string) (*types.Domain, error) {
	domain, err := d.domainRepository.Find(ctx, name)
	if err != nil {
		return nil, err
	}

	if domain.ApplicationID != applicationID {
		return nil, errors.New("access denied")
	}

	return domain, d.domainRepository.Delete(ctx, domain.ID)
}

func (d *domainService) FindByApplicationID(ctx context.Context, applicationID uuid.UUID) ([]*types.Domain, error) {
	return d.domainRepository.FindByApplicationID(ctx, applicationID)
}

func (d *domainService) FindForEnvironmentAndInstanceType(ctx context.Context, applicationID uuid.UUID, environment string, ist types.InstanceType) ([]*types.Domain, error) {
	all, err := d.FindByApplicationID(ctx, applicationID)
	if err != nil {
		return nil, err
	}

	return lo.Filter(all, func(item *types.Domain, index int) bool {
		return item.Environment == environment && item.InstanceType == ist
	}), nil
}
