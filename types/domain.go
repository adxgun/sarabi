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
	// sarabi domain add dev-api.usemoyo.app --env=dev --instance=backend
	// 0.9: validate duplicate
	// 1. get active deployment application_id and environment and instance_type
	// 2. add the domain to the []host matcher for that particular deployment(can be found by comparing the deployment access-url to match.host[])

	// sarabi domain remove dev-api.usemoyo.app
	// 1. get all domains in the caddy config
	// 2. find this specific one, remove it from match.host[]
	// 3. update it to deleted in the database
)
