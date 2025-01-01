package types

import (
	"github.com/google/uuid"
	"time"
)

type (
	Log struct {
		ID            uuid.UUID `gorm:"primaryKey" json:"id"`
		DeploymentID  uuid.UUID `json:"deployment_id"`
		ApplicationID uuid.UUID `json:"application_id"`
		Environment   string    `json:"environment"`
		Location      string    `json:"location"`
		StorageType   string    `json:"storage_type"`
		ContainerID   string    `json:"container_id"`
		Timestamp     time.Time `json:"created_at"`
	}

	LogEntry struct {
		Owner string `json:"owner"`
		Log   string `json:"log"`
	}

	Filter struct {
		Environment string
		From        time.Time
		To          time.Time
	}
)
