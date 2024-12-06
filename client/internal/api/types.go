package api

import "github.com/google/uuid"

type (
	DeployParams struct {
		Instances     int       `json:"instances"`
		ApplicationID uuid.UUID `json:"application_id"`
		Environment   string    `json:"environment"`
		StorageEngine string    `json:"storage_engine"`
	}

	CreateApplicationParams struct {
		Name          string `json:"name" validate:"required"`
		Domain        string `json:"domain" validate:"required,fqdn"`
		StorageEngine string `json:"storage_engine" validate:"required"`
		FePath        string
		BePath        string
	}
)

type (
	Application struct {
		ID             uuid.UUID `json:"id"`
		Name           string    `json:"name"`
		Domain         string    `json:"domain"`
		StorageEngines []string  `json:"storage_engines"`
	}
)
