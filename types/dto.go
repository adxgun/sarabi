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
)
