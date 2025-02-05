package postgres

import (
	"fmt"
	"sarabi/internal/misc"
	types "sarabi/internal/types"
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
	password, _ := misc.DefaultRandomIdGenerator.Generate(64)
	dbName := fmt.Sprintf("postgres-%s-%s", dep.Application.Name, dep.Environment)
	username := fmt.Sprintf("%s-%s-user", dep.Application.Name, dep.Environment)
	host := fmt.Sprintf("postgres-%s-%s", dep.Application.Name, dep.Environment)
	databaseUrl := misc.FormatURI("postgres", username, password, host, p.Port(), dbName, "disable")
	return []types.CreateSecretParams{
		{Key: "POSTGRES_DB", Value: dbName, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "POSTGRES_USER", Value: username, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "POSTGRES_HOST", Value: host, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "POSTGRES_PORT", Value: p.Port(), Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "POSTGRES_PASSWORD", Value: password, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "POSTGRES_DATABASE_URL", Value: databaseUrl, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
	}
}

func (p postgresProvider) DataPath() string {
	return "/var/lib/postgresql/data"
}

func (p postgresProvider) Port() string {
	return "5432"
}

func (p postgresProvider) Engine() types.StorageEngine {
	return types.StorageEnginePostgres
}
