package mongo

import (
	"fmt"
	"github.com/google/uuid"
	types2 "sarabi/internal/types"
)

type mongoProvider struct{}

func New() *mongoProvider {
	return &mongoProvider{}
}

func (p mongoProvider) ContainerName(dep *types2.Deployment) string {
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

func (p mongoProvider) EnvVars(dep *types2.Deployment) []types2.CreateSecretParams {
	return []types2.CreateSecretParams{
		{Key: "MONGO_INITDB_ROOT_USERNAME", Value: fmt.Sprintf("%s-%s-user", dep.Application.Name, dep.Environment), Environment: dep.Environment, InstanceType: types2.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MONGO_HOST", Value: fmt.Sprintf("mysql-%s-%s", dep.Application.Name, dep.Environment), Environment: dep.Environment, InstanceType: types2.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MONGO_PORT", Value: "27017", Environment: dep.Environment, InstanceType: types2.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MONGO_INITDB_ROOT_PASSWORD", Value: uuid.NewString(), Environment: dep.Environment, InstanceType: types2.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
	}
}

func (p mongoProvider) DataPath() string {
	return "/data/db"
}

func (p mongoProvider) Port() string {
	return "27017"
}
