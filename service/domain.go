package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	proxycomponent "sarabi/components/proxy"
	"sarabi/database"
	"sarabi/integrations/caddy"
	"sarabi/types"
)

type (
	DomainService interface {
		AddDomain(ctx context.Context, applicationID uuid.UUID, params types.AddDomainParams) (*types.Domain, error)
		RemoveDomain(ctx context.Context, applicationID uuid.UUID, name string) error
	}

	domainService struct {
		caddyClient        caddy.Client
		domainRepository   database.DomainRepository
		applicationService ApplicationService
	}
)

func NewDomainService(caddyClient caddy.Client, domainRepo database.DomainRepository, appService ApplicationService) DomainService {
	return &domainService{
		caddyClient:        caddyClient,
		domainRepository:   domainRepo,
		applicationService: appService,
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

	deployment, err := d.applicationService.FindCurrentlyActiveDeploymentsEnv(ctx, applicationID,
		params.InstanceType, params.Environment)
	if err != nil {
		return nil, err
	}

	err = d.caddyClient.ApplyDomainConfig(ctx, proxycomponent.ProxyServerConfigUrl, newDomain, deployment, types.DomainOperationAdd)
	if err != nil {
		return nil, err
	}
	return newDomain, nil
}

func (d *domainService) RemoveDomain(ctx context.Context, applicationID uuid.UUID, name string) error {
	//TODO implement me
	panic("implement me")
}
