package database

import (
	"context"
	"github.com/google/uuid"
	types2 "sarabi/internal/types"
)

type ApplicationRepository interface {
	Save(ctx context.Context, app *types2.Application) error
	FindByID(ctx context.Context, id uuid.UUID) (*types2.Application, error)
	FindByName(ctx context.Context, name string) (*types2.Application, error)
	FindAll(ctx context.Context) ([]*types2.Application, error)
}

type SecretRepository interface {
	Save(ctx context.Context, secret *types2.Secret) error
	FindAll(ctx context.Context, applicationID uuid.UUID) ([]*types2.Secret, error)
	FindBy(ctx context.Context, applicationID uuid.UUID, name, env, instanceType string) (*types2.Secret, error)
	UpdateValue(ctx context.Context, id uuid.UUID, newValue string) error
}

type DeploymentSecretRepository interface {
	SaveAll(ctx context.Context, applicationSecrets []*types2.DeploymentSecret) error
	FindAll(ctx context.Context, deploymentID uuid.UUID) ([]*types2.DeploymentSecret, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type DeploymentRepository interface {
	Save(ctx context.Context, deployment *types2.Deployment) error
	FindAll(ctx context.Context, applicationID uuid.UUID) ([]*types2.Deployment, error)
	FindByID(ctx context.Context, deploymentID uuid.UUID) (*types2.Deployment, error)
	UpdateDeploymentStatus(ctx context.Context, deploymentID uuid.UUID, newStatus string) error
	FindByIdentifier(ctx context.Context, identifier string) ([]*types2.Deployment, error)
}

type DomainRepository interface {
	Save(ctx context.Context, domain *types2.Domain) error
	FindByID(ctx context.Context, id uuid.UUID) (*types2.Domain, error)
	Find(ctx context.Context, name string) (*types2.Domain, error)
	Delete(ctx context.Context, id uuid.UUID) error
	FindByApplicationID(ctx context.Context, applicationID uuid.UUID) ([]*types2.Domain, error)
}

type BackupSettingsRepository interface {
	Save(ctx context.Context, settings *types2.BackupSettings) error
	FindAll(ctx context.Context) ([]*types2.BackupSettings, error)
	FindByApplicationID(ctx context.Context, applicationID uuid.UUID) ([]*types2.BackupSettings, error)
}

type CredentialRepository interface {
	Save(ctx context.Context, cred *types2.Credential) error
	FindByApplicationID(ctx context.Context, applicationID uuid.UUID) ([]*types2.Credential, error)
	UpdateCredentialValue(ctx context.Context, id uuid.UUID, newValue string) error
	FindByName(ctx context.Context, applicationID uuid.UUID, provider, key string) (*types2.Credential, error)
}

type BackupRepository interface {
	Save(ctx context.Context, bc *types2.Backup) error
	FindByApplicationID(ctx context.Context, applicationID uuid.UUID) ([]*types2.Backup, error)
	FindByID(ctx context.Context, id uuid.UUID) (*types2.Backup, error)
}
