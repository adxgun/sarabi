package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sarabi/database"
	"sarabi/integrations/caddy"
	"sarabi/types"
)

type (
	DomainService interface {
		AddDomain(ctx context.Context, applicationID uuid.UUID, params types.AddDomainParams) (*types.Domain, error)
		RemoveDomain(ctx context.Context, applicationID uuid.UUID, name string) (*types.Domain, error)
		FindByApplicationID(ctx context.Context, applicationID uuid.UUID) ([]*types.Domain, error)
	}

	domainService struct {
		caddyClient      caddy.Client
		domainRepository database.DomainRepository
	}
)

func NewDomainService(caddyClient caddy.Client, domainRepo database.DomainRepository) DomainService {
	return &domainService{
		caddyClient:      caddyClient,
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
