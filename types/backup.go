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
)
