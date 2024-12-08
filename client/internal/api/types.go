package api

import (
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
		Name          string `json:"name" validate:"required"`
		Domain        string `json:"domain" validate:"required,fqdn"`
		StorageEngine string `json:"storage_engine" validate:"required"`
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
)
