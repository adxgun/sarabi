package types

import (
	"fmt"
	"github.com/google/uuid"
	"time"
)

const (
	ServerConfigObjectStorage = "object_storage"
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

	ServerConfig struct {
		ID            uuid.UUID `gorm:"primaryKey"`
		ApplicationID uuid.UUID `gorm:"not null"`
		Provider      string    `json:"provider"`
		Name          string
		Value         string
		CreatedAt     time.Time
	}

	AddCredentialsParams struct {
		ApplicationID uuid.UUID          `json:"application_id"`
		Provider      string             `json:"provider"`
		Value         StorageCredentials `json:"value"`
	}

	StorageCredentials struct {
		AccessKeyID string `json:"access_key_id"`
		SecretKey   string `json:"secret_key"`
		Endpoint    string `json:"endpoint"`
		Region      string `json:"region"`
	}

	AddCredentialsResponse struct {
		ID       uuid.UUID `json:"id"`
		Provider string    `json:"provider"`
		Key      string    `json:"key"`
	}

	VarResponse struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		Value       string    `json:"value"`
		Environment string    `json:"environment"`
	}

	CreateServerConfigParams struct {
		ApplicationID uuid.UUID
		Name          string
		Provider      string
		Value         any
	}

	ServerConfigResponse struct {
		ID uuid.UUID `json:"id"`
	}
)

func (s *Secret) Env() string {
	return fmt.Sprintf("%s=%s", s.Name, s.Value)
}

func (s *StorageCredentials) URI() string {
	return fmt.Sprintf("s3://%s:%s@%s/sarabi-logs", s.AccessKeyID, s.SecretKey, s.Endpoint)
}
