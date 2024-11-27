package types

import (
	"fmt"
	"github.com/google/uuid"
	"time"
)

const (
	CredentialProviderS3 = "s3"
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

	Credential struct {
		ID            uuid.UUID `gorm:"primaryKey"`
		ApplicationID uuid.UUID `gorm:"not null"`
		Provider      string    `json:"provider"`
		Name          string
		Value         string
		CreatedAt     time.Time
	}

	AddCredentialsParams struct {
		ApplicationID uuid.UUID `json:"application_id"`
		Provider      string    `json:"provider"`
		Values        []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"values"`
	}

	AddCredentialsResponse struct {
		ID       uuid.UUID `json:"id"`
		Provider string    `json:"provider"`
		Key      string    `json:"key"`
	}
)

func (s *Secret) Env() string {
	return fmt.Sprintf("%s=%s", s.Name, s.Value)
}
