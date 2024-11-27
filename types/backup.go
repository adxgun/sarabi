package types

import (
	"github.com/google/uuid"
	"time"
)

type (
	StorageCredentials struct {
		Endpoint, KeyId, SecretKey, Region string
	}

	BackupSettings struct {
		ID             uuid.UUID `gorm:"primaryKey"`
		ApplicationID  uuid.UUID
		Environment    string
		BackupInterval time.Duration
		CreatedAt      time.Time
		DeletedAt      time.Time
	}

	Backup struct {
		ID            uuid.UUID     `json:"id" gorm:"primaryKey"`
		ApplicationID uuid.UUID     `json:"application_id"`
		Environment   string        `json:"environment"`
		CreatedAt     time.Time     `json:"created_at"`
		StorageEngine StorageEngine `json:"storage_engine"`
		Location      string        `json:"location"`
		StorageType   string        `json:"storage_type"`

		Application *Application `gorm:"foreignKey:ApplicationID"`
	}
)
