package databasecomponent

import (
	"go.uber.org/zap"
	"sarabi/internal/components/database/providers/mongo"
	"sarabi/internal/components/database/providers/mysql"
	"sarabi/internal/components/database/providers/postgres"
	types "sarabi/internal/types"
	"sarabi/logger"
	"strings"
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
	logger.Info("getting provider for engine",
		zap.Any("engine", engine))
	switch types.StorageEngine(strings.ToLower(string(engine))) {
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
