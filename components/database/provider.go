package databasecomponent

import (
	"sarabi/components/database/providers/mongo"
	"sarabi/components/database/providers/mysql"
	"sarabi/components/database/providers/postgres"
	"sarabi/types"
)

type (
	Provider interface {
		ContainerName(dep *types.Deployment) string

		Image() string

		Setup() error

		EnvVars(dep *types.Deployment) []types.CreateSecretParams

		DataPath() string

		Port() string
	}
)

func NewProvider(engine types.StorageEngine) Provider {
	switch engine {
	case types.StorageEnginePostgres:
		return postgres.New()
	case types.StorageEngineMysql:
		return mysql.New()
	case types.StorageEngineMongo:
		return mongo.New()
	default:
		panic("unsupported engine " + string(engine))
	}
}
