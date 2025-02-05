package mysql

import (
	"fmt"
	"sarabi/internal/misc"
	types "sarabi/internal/types"
)

type mysqlProvider struct{}

func New() *mysqlProvider {
	return &mysqlProvider{}
}

func (p mysqlProvider) ContainerName(dep *types.Deployment) string {
	return fmt.Sprintf("mysql-%s-%s", dep.Application.Name, dep.Environment)
}

func (p mysqlProvider) Image() string {
	return "mysql:9.1.0"
}

func (p mysqlProvider) Setup() error {
	// TODO:
	// 1. write mysql config
	return nil
}

func (p mysqlProvider) EnvVars(dep *types.Deployment) []types.CreateSecretParams {
	masterPassword, _ := misc.DefaultRandomIdGenerator.Generate(100)
	password, _ := misc.DefaultRandomIdGenerator.Generate(64)
	dbName := fmt.Sprintf("mysql-%s-%s", dep.Application.Name, dep.Environment)
	username := fmt.Sprintf("%s-%s-user", dep.Application.Name, dep.Environment)
	host := fmt.Sprintf("mysql-%s-%s", dep.Application.Name, dep.Environment)
	databaseUrl := misc.FormatURI("mysql", username, password, host, p.Port(), dbName, "disable")
	return []types.CreateSecretParams{
		{Key: "MYSQL_DATABASE", Value: dbName, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MYSQL_USER", Value: username, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MYSQL_HOST", Value: host, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MYSQL_PORT", Value: p.Port(), Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MYSQL_ROOT_PASSWORD", Value: masterPassword, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MYSQL_PASSWORD", Value: password, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MYSQL_DATABASE_URL", Value: databaseUrl, Environment: dep.Environment, InstanceType: types.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
	}
}

func (p mysqlProvider) DataPath() string {
	return "/var/lib/mysql"
}

func (p mysqlProvider) Port() string {
	return "3306"
}

func (p mysqlProvider) Engine() types.StorageEngine {
	return types.StorageEngineMysql
}
