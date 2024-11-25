package mongo

import (
	"fmt"
	"github.com/google/uuid"
	"sarabi/types"
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
	return []types.CreateSecretParams{
		{Key: "MONGO_INITDB_ROOT_USERNAME", Value: fmt.Sprintf("%s-%s-user", dep.Application.Name, dep.Environment), Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MONGO_HOST", Value: fmt.Sprintf("mysql-%s-%s", dep.Application.Name, dep.Environment), Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MONGO_PORT", Value: "27017", Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MONGO_INITDB_ROOT_PASSWORD", Value: uuid.NewString(), Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
	}
}

func (p mongoProvider) DataPath() string {
	return "/data/db"
}

func (p mongoProvider) Port() string {
	return "27017"
}
