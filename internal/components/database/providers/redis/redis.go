package redis

import (
	"fmt"
	"github.com/google/uuid"
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
	return "sarabi-redis:7"
}

func (p redisProvider) Setup() error {
	// TODO:
	// 1. write mongo config
	return nil
}

func (p redisProvider) EnvVars(dep *types.Deployment) []types.CreateSecretParams {
	return []types.CreateSecretParams{
		{Key: "REDIS_USER", Value: fmt.Sprintf("%s-%s-user", dep.Application.Name, dep.Environment), Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "REDIS_HOST", Value: fmt.Sprintf("redis-%s-%s", dep.Application.Name, dep.Environment), Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "REDIS_PORT", Value: "6379", Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "REDIS_PASSWORD", Value: uuid.NewString(), Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
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
