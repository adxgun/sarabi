package types

import "github.com/google/uuid"

type DomainOperation string

var (
	DomainOperationAdd    DomainOperation = "add"
	DomainOperationRemove DomainOperation = "remove"
)

type (
	Domain struct {
		ID            uuid.UUID
		ApplicationID uuid.UUID
		Name          string
		Environment   string
		InstanceType  InstanceType
		Status        string

		Application Application `gorm:"foreignKey:ApplicationID"`
	}
)
