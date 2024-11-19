package postgres

import (
	"fmt"
	"github.com/google/uuid"
	"sarabi/types"
)

type postgresProvider struct{}

func New() *postgresProvider {
	return &postgresProvider{}
}

func (p postgresProvider) ContainerName(dep *types.Deployment) string {
	return fmt.Sprintf("postgres-%s-%s", dep.Application.Name, dep.Environment)
}

func (p postgresProvider) Image() string {
	return "postgres:17"
}

func (p postgresProvider) Setup() error {
	// TODO:
	// 1. write postgresql config type
	//  PostgresSQLConfig struct {
	//    MaxConnections      int // 300
	//    SharedBuffers       string // 30% of total memory
	//    WorkMem             string // 16MB
	//    MaintenanceWorkMem  string // 200MB
	//}
	return nil
}

func (p postgresProvider) EnvVars(dep *types.Deployment) []types.CreateSecretParams {
	return []types.CreateSecretParams{
		{Key: "POSTGRES_DB", Value: fmt.Sprintf("postgres-%s-%s", dep.Application.Name, dep.Environment), Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "POSTGRES_USER", Value: fmt.Sprintf("%s-%s-user", dep.Application.Name, dep.Environment), Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "POSTGRES_HOST", Value: fmt.Sprintf("postgres-%s-%s", dep.Application.Name, dep.Environment), Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "POSTGRES_PORT", Value: "5432", Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "POSTGRES_PASSWORD", Value: uuid.NewString(), Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
	}
}

func (p postgresProvider) DataPath() string {
	return "/var/lib/postgresql/data"
}

func (p postgresProvider) Port() string {
	return "5432"
}
