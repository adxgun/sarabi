package types

import (
	"github.com/google/uuid"
)

type (
	CreateSecretParams struct {
		Key           string       `json:"key"`
		Value         string       `json:"value"`
		Environment   string       `json:"environment"`
		InstanceType  InstanceType `json:"instance_type"`
		ApplicationID uuid.UUID    `json:"application_id"`
	}

	AddDomainParams struct {
		Name         string       `json:"name"`
		InstanceType InstanceType `json:"instance_type"`
		Environment  string       `json:"environment"`
	}

	DeployResponse struct {
		Identifier string    `json:"identifier"`
		AccessURL  AccessURL `json:"access_url"`
	}

	AccessURL struct {
		Frontend []string `json:"frontend"`
		Backend  []string `json:"backend"`
	}
)
