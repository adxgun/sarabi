package redis

import (
	"fmt"
	"sarabi/internal/misc"
	"sarabi/internal/types"
)

type redisProvider struct{}

func New() *redisProvider {
	return &redisProvider{}
}

func (p redisProvider) ContainerName(dep *types.Deployment) string {
	return fmt.Sprintf("redis-%s-%s", dep.Application.Name, dep.Environment)
}

func (p redisProvider) Image() string {
	return "bitnami/redis:7.4"
}

func (p redisProvider) Setup() error {
	// TODO:
	// 1. write mongo config
	return nil
}

func (p redisProvider) EnvVars(dep *types.Deployment) []types.CreateSecretParams {
	password, _ := misc.DefaultRandomIdGenerator.Generate(64)
	dbName := fmt.Sprintf("redis-%s-%s", dep.Application.Name, dep.Environment)
	username := fmt.Sprintf("%s-%s-user", dep.Application.Name, dep.Environment)
	host := fmt.Sprintf("redis-%s-%s", dep.Application.Name, dep.Environment)
	databaseUrl := misc.FormatURI("redis", username, password, host, p.Port(), dbName, "disable")
	return []types.CreateSecretParams{
		{Key: "REDIS_USER", Value: username, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "REDIS_HOST", Value: host, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "REDIS_PORT", Value: p.Port(), Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "REDIS_PASSWORD", Value: password, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "REDIS_URL", Value: databaseUrl, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
	}
}

func (p redisProvider) DataPath() string {
	return "/data"
}

func (p redisProvider) Port() string {
	return "6379"
}

func (p redisProvider) Engine() types.StorageEngine {
	return types.StorageEngineMongo
}
