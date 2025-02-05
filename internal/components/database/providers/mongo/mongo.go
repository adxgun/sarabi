package mongo

import (
	"fmt"
	"sarabi/internal/misc"
	types "sarabi/internal/types"
)

type mongoProvider struct{}

func New() *mongoProvider {
	return &mongoProvider{}
}

func (p mongoProvider) ContainerName(dep *types.Deployment) string {
	return fmt.Sprintf("mongo-%s-%s", dep.Application.Name, dep.Environment)
}

func (p mongoProvider) Image() string {
	return "mongo:8.0.3"
}

func (p mongoProvider) Setup() error {
	// TODO:
	// 1. write mongo config
	return nil
}

func (p mongoProvider) EnvVars(dep *types.Deployment) []types.CreateSecretParams {
	password, _ := misc.DefaultRandomIdGenerator.Generate(64)
	dbName := fmt.Sprintf("mongo-%s-%s", dep.Application.Name, dep.Environment)
	username := fmt.Sprintf("%s-%s-user", dep.Application.Name, dep.Environment)
	host := fmt.Sprintf("mongo-%s-%s", dep.Application.Name, dep.Environment)
	databaseUrl := misc.FormatURI("mongo", username, password, host, p.Port(), dbName, "disable")
	return []types.CreateSecretParams{
		{Key: "MONGO_INITDB_ROOT_USERNAME", Value: username, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MONGO_HOST", Value: host, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MONGO_PORT", Value: p.Port(), Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MONGO_INITDB_ROOT_PASSWORD", Value: password, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MONGO_DATABASE_URL", Value: databaseUrl, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
	}
}

func (p mongoProvider) DataPath() string {
	return "/data/db"
}

func (p mongoProvider) Port() string {
	return "27017"
}

func (p mongoProvider) Engine() types.StorageEngine {
	return types.StorageEngineMongo
}
