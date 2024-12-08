package mysql

import (
	"fmt"
	"github.com/google/uuid"
	types2 "sarabi/internal/types"
)

type mysqlProvider struct{}

func New() *mysqlProvider {
	return &mysqlProvider{}
}

func (p mysqlProvider) ContainerName(dep *types2.Deployment) string {
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

func (p mysqlProvider) EnvVars(dep *types2.Deployment) []types2.CreateSecretParams {
	return []types2.CreateSecretParams{
		{Key: "MYSQL_DATABASE", Value: fmt.Sprintf("mysql-%s-%s", dep.Application.Name, dep.Environment), Environment: dep.Environment, InstanceType: types2.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MYSQL_USER", Value: fmt.Sprintf("%s-%s-user", dep.Application.Name, dep.Environment), Environment: dep.Environment, InstanceType: types2.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MYSQL_HOST", Value: fmt.Sprintf("mysql-%s-%s", dep.Application.Name, dep.Environment), Environment: dep.Environment, InstanceType: types2.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MYSQL_PORT", Value: "3306", Environment: dep.Environment, InstanceType: types2.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MYSQL_ROOT_PASSWORD", Value: uuid.NewString(), Environment: dep.Environment, InstanceType: types2.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
		{Key: "MYSQL_PASSWORD", Value: uuid.NewString(), Environment: dep.Environment, InstanceType: types2.InstanceTypeDatabase, ApplicationID: dep.ApplicationID},
	}
}

func (p mysqlProvider) DataPath() string {
	return "/var/lib/mysql"
}

func (p mysqlProvider) Port() string {
	return "3306"
}
