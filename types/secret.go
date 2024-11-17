package types

import (
	"fmt"
	"github.com/google/uuid"
	"time"
)

type (
	Secret struct {
		ID            uuid.UUID `gorm:"primaryKey"`
		ApplicationID uuid.UUID
		Name          string
		Value         string
		Environment   string
		InstanceType  string
		CreatedAt     time.Time
		UpdatedAt     time.Time
		DeletedAt     time.Time
	}

	DeploymentSecret struct {
		ID           uuid.UUID  `gorm:"primaryKey"`
		DeploymentID uuid.UUID  `gorm:"not null"`
		SecretID     uuid.UUID  `gorm:"not null"`
		Deployment   Deployment `gorm:"foreignKey:DeploymentID"`
		Secret       Secret     `gorm:"foreignKey:SecretID"`
	}
)

func (s *Secret) Env() string {
	return fmt.Sprintf("%s=%s", s.Name, s.Value)
}

func DBSecrets(deployment *Deployment) []CreateSecretParams {
	return []CreateSecretParams{
		{Key: "POSTGRES_DB", Value: deployment.Application.Name, InstanceType: deployment.InstanceType, Environment: deployment.Environment, ApplicationID: deployment.ApplicationID},
		{Key: "POSTGRES_PASSWORD", Value: uuid.New().String(), InstanceType: deployment.InstanceType, Environment: deployment.Environment, ApplicationID: deployment.ApplicationID},
		{Key: "POSTGRES_USER", Value: deployment.Application.Name + "-user", InstanceType: deployment.InstanceType, Environment: deployment.Environment, ApplicationID: deployment.ApplicationID},
		{Key: "DATABASE_HOST", Value: deployment.DBInstanceName(), InstanceType: deployment.InstanceType, Environment: deployment.Environment, ApplicationID: deployment.ApplicationID},
		{Key: "DATABASE_PORT", Value: "5432", InstanceType: deployment.InstanceType, Environment: deployment.Environment, ApplicationID: deployment.ApplicationID},
	}
}
