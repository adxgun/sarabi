package databasecomponent

import (
	"go.uber.org/zap"
	"sarabi/internal/components/database/providers/mongo"
	"sarabi/internal/components/database/providers/mysql"
	"sarabi/internal/components/database/providers/postgres"
	types2 "sarabi/internal/types"
	"sarabi/logger"
	"strings"
)

type (
	Provider interface {
		ContainerName(dep *types2.Deployment) string

		Image() string

		Setup() error

		EnvVars(dep *types2.Deployment) []types2.CreateSecretParams

		DataPath() string

		Port() string
	}
)

func NewProvider(engine types2.StorageEngine) Provider {
	logger.Info("getting provider for engine",
		zap.Any("engine", engine))
	switch types2.StorageEngine(strings.ToLower(string(engine))) {
	case types2.StorageEnginePostgres:
		return postgres.New()
	case types2.StorageEngineMysql:
		return mysql.New()
	case types2.StorageEngineMongo:
		return mongo.New()
	default:
		panic("unsupported engine " + string(engine))
	}
}
