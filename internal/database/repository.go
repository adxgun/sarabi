package database

import (
	"context"
	"github.com/google/uuid"
	"sarabi/internal/types"
)

type ApplicationRepository interface {
	Save(ctx context.Context, app *types.Application) error
	FindByID(ctx context.Context, id uuid.UUID) (*types.Application, error)
	FindByName(ctx context.Context, name string) (*types.Application, error)
	FindAll(ctx context.Context) ([]*types.Application, error)
}

type SecretRepository interface {
	Save(ctx context.Context, secret *types.Secret) error
	FindAll(ctx context.Context, applicationID uuid.UUID) ([]*types.Secret, error)
	FindBy(ctx context.Context, applicationID uuid.UUID, name, env, instanceType string) (*types.Secret, error)
	UpdateValue(ctx context.Context, id uuid.UUID, newValue string) error
}

type DeploymentSecretRepository interface {
	SaveAll(ctx context.Context, applicationSecrets []*types.DeploymentSecret) error
	FindAll(ctx context.Context, deploymentID uuid.UUID) ([]*types.DeploymentSecret, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type DeploymentRepository interface {
	Save(ctx context.Context, deployment *types.Deployment) error
	FindAll(ctx context.Context, applicationID uuid.UUID) ([]*types.Deployment, error)
	FindByID(ctx context.Context, deploymentID uuid.UUID) (*types.Deployment, error)
	UpdateDeploymentStatus(ctx context.Context, deploymentID uuid.UUID, newStatus string) error
	FindByIdentifier(ctx context.Context, identifier string) ([]*types.Deployment, error)
}

type DomainRepository interface {
	Save(ctx context.Context, domain *types.Domain) error
	FindByID(ctx context.Context, id uuid.UUID) (*types.Domain, error)
	Find(ctx context.Context, name string) (*types.Domain, error)
	Delete(ctx context.Context, id uuid.UUID) error
	FindByApplicationID(ctx context.Context, applicationID uuid.UUID) ([]*types.Domain, error)
}

type BackupSettingsRepository interface {
	Save(ctx context.Context, settings *types.BackupSettings) error
	FindAll(ctx context.Context) ([]*types.BackupSettings, error)
	FindByApplicationID(ctx context.Context, applicationID uuid.UUID) ([]*types.BackupSettings, error)
	UpdateExpression(ctx context.Context, id uuid.UUID, cronExpression string) error
}

type ServerConfigRepository interface {
	Save(ctx context.Context, cred *types.ServerConfig) error
	FindByApplicationID(ctx context.Context, applicationID uuid.UUID) ([]*types.ServerConfig, error)
	UpdateServerConfigValue(ctx context.Context, id uuid.UUID, newValue string) error
	FindByName(ctx context.Context, applicationID uuid.UUID, provider, name string) (*types.ServerConfig, error)
}

type BackupRepository interface {
	Save(ctx context.Context, bc *types.Backup) error
	FindByApplicationID(ctx context.Context, applicationID uuid.UUID) ([]*types.Backup, error)
	FindByID(ctx context.Context, id uuid.UUID) (*types.Backup, error)
}

type NetworkAccessRepository interface {
	Save(ctx context.Context, na *types.NetworkAccess) error
	FindByApplicationID(ctx context.Context, applicationID uuid.UUID) ([]*types.NetworkAccess, error)
	FindByIP(ctx context.Context, ip string) ([]*types.NetworkAccess, error)
	Remove(ctx context.Context, naID uuid.UUID) error
}

type LogsRepository interface {
	Save(ctx context.Context, log *types.Log) error
	FindAll(ctx context.Context, applicationID uuid.UUID, filter types.Filter) ([]*types.Log, error)
}
