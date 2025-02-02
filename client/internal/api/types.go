package api

import (
	"encoding/json"
	"github.com/google/uuid"
	"time"
)

type (
	DeployParams struct {
		Instances     int       `json:"instances"`
		ApplicationID uuid.UUID `json:"application_id"`
		Environment   string    `json:"environment" validate:"required"`
	}

	CreateApplicationParams struct {
		Name          string   `json:"name" validate:"required"`
		Domain        string   `json:"domain" validate:"required,fqdn"`
		StorageEngine []string `json:"storage_engine" validate:"required"`
		FePath        string
		BePath        string
	}

	UpdateVariablesParams struct {
		Environment string `json:"environment"`
		Vars        []KV   `json:"vars"`
	}

	KV struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	AddDomainParam struct {
		FQDN        string `json:"name" validate:"required,fqdn"`
		Instance    string `json:"instance_type" validate:"required"`
		Environment string `json:"environment" validate:"required"`
	}

	RemoveDomainParam struct {
		FQDN string `json:"name" validate:"required,fqdn"`
	}

	ScaleAppParams struct {
		Count       int    `json:"count"`
		Environment string `json:"environment"`
	}

	RollbackParams struct {
		Identifier string `json:"identifier"`
	}

	CreateBackupParams struct {
		Environment    string `json:"environment"`
		CronExpression string `json:"cron_expression"`
	}
)

type (
	Application struct {
		ID             uuid.UUID `json:"id"`
		Name           string    `json:"name"`
		Domain         string    `json:"domain"`
		StorageEngines []string  `json:"storage_engines"`
		CreatedAt      time.Time `json:"created_at"`
	}

	DeployResponse struct {
		Identifier string    `json:"identifier"`
		AccessURL  AccessURL `json:"access_url"`
	}

	AccessURL struct {
		Frontend []string `json:"frontend"`
		Backend  []string `json:"backend"`
	}

	Var struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Value       string `json:"value"`
		Environment string `json:"environment"`
	}

	Deployment struct {
		ID            uuid.UUID `json:"id"`
		ApplicationID uuid.UUID `json:"applicationID"`
		Environment   string    `json:"environment"`
		Status        string    `json:"status"`
		Instances     int       `json:"instances"`
		Name          string    `json:"name"`
		Port          string    `json:"port"`
		InstanceType  string    `json:"instance_type"`
		Identifier    string    `json:"identifier"`
		CreatedAt     time.Time `json:"created_at"`
	}

	Backup struct {
		ID            uuid.UUID `json:"id" gorm:"primaryKey"`
		ApplicationID uuid.UUID `json:"application_id"`
		Environment   string    `json:"environment"`
		CreatedAt     time.Time `json:"created_at"`
		StorageEngine string    `json:"storage_engine"`
		Location      string    `json:"location"`
		StorageType   string    `json:"storage_type"`
	}

	LogEntry struct {
		Owner string `json:"owner"`
		Log   string `json:"log"`
	}

	Event struct {
		Type    Type            `json:"type"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}

	Type string
)

const (
	Error    Type = "error"
	Info     Type = "info"
	Success  Type = "success"
	Complete Type = "complete"
)

func (b Backup) StorageTypeString() string {
	switch b.StorageType {
	case "S3":
		return "Object Storage"
	case "File":
		return "File Storage"
	}
	return "Unknown"
}
