package database

import (
	"context"
	"github.com/google/uuid"
	"sarabi/types"
)

type ApplicationRepository interface {
	Save(ctx context.Context, app *types.Application) error
	FindByID(ctx context.Context, id uuid.UUID) (*types.Application, error)
	FindByName(ctx context.Context, name string) (*types.Application, error)
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
}

type DeploymentRepository interface {
	Save(ctx context.Context, deployment *types.Deployment) error
	FindAll(ctx context.Context, applicationID uuid.UUID) ([]*types.Deployment, error)
	FindByID(ctx context.Context, deploymentID uuid.UUID) (*types.Deployment, error)
	UpdateDeploymentStatus(ctx context.Context, deploymentID uuid.UUID, newStatus string) error
	FindByIdentifier(ctx context.Context, identifier string) ([]*types.Deployment, error)
}
