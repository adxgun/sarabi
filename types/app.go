package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"io"
	"strings"
	"time"
)

var (
	sarabiDataPath = "/var/sarabi/data"
)

type (
	Application struct {
		ID             uuid.UUID      `gorm:"primaryKey"`
		Name           string         `json:"name"`
		Domain         string         `json:"domain"`
		StorageEngines StorageEngines `json:"storage_engines"`
		CreatedAt      time.Time
		UpdatedAt      time.Time      `json:"-"`
		DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	}

	Deployment struct {
		ID            uuid.UUID    `gorm:"primaryKey" json:"id"`
		ApplicationID uuid.UUID    `gorm:"not null" json:"applicationID"`
		Environment   string       `json:"environment"`
		Status        string       `json:"status"`
		Instances     int          `json:"instances"`
		Port          string       `json:"port"`
		InstanceType  InstanceType `json:"instance_type"` // frontend, backend, database, proxy
		Identifier    string       `json:"identifier"`
		Application   Application  `gorm:"foreignKey:ApplicationID" json:"-"`
		CreatedAt     time.Time    `json:"created_at"`
	}

	DeploymentStatus string
	InstanceType     string
)

type (
	CreateApplicationParams struct {
		Name          string `json:"name" binding:"required"`
		Domain        string `json:"domain" binding:"required"`
		StorageEngine string `json:"storage_engine" binding:"required"`
	}

	DeployParams struct {
		ApplicationID uuid.UUID
		Frontend      io.Reader
		Backend       io.Reader
		Instances     int
		Environment   string
		StorageEngine StorageEngine
	}

	CreateDeploymentParams struct {
		ApplicationID uuid.UUID `json:"application_id"`
		Environment   string    `json:"environment"`
		Instances     int
		Port          string       `json:"-"`
		InstanceType  InstanceType `json:"instance_type"` // frontend, backend, database, proxy
		Identifier    string       `json:"identifier"`
	}
)

type StorageEngine string
type StorageEngines []StorageEngine

const (
	StorageEnginePostgres StorageEngine = "postgres"
	StorageEngineMysql    StorageEngine = "mysql"
	StorageEngineMongo    StorageEngine = "mongo"
)

const (
	DeploymentStatusActive  DeploymentStatus = "ACTIVE"
	DeploymentStatusCreated DeploymentStatus = "CREATED"
	DeploymentStatusStopped DeploymentStatus = "STOPPED"
)

func (s StorageEngine) Value() (driver.Value, error) {
	return string(s), nil
}

func (s *StorageEngine) Scan(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return errors.New("failed to scan StorageEngine: type assertion to string failed")
	}

	*s = StorageEngine(v)
	return nil
}

func (s StorageEngines) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *StorageEngines) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan []StorageEngine: type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, s)
}

const (
	InstanceTypeFrontend InstanceType = "frontend"
	InstanceTypeBackend  InstanceType = "backend"
	InstanceTypeProxy    InstanceType = "proxy"
	InstanceTypeDatabase InstanceType = "database"
)

func (a *Deployment) ImageName() string {
	return fmt.Sprintf("%s:%s", strings.ReplaceAll(a.ID.String(), "-", ""), a.Environment)
}

func (a *Deployment) DBInstanceName() string {
	return fmt.Sprintf("postgres-%s", a.Application.Name)
}

func (a *Deployment) NetworkName() string {
	return fmt.Sprintf("network-%s-%s", strings.ReplaceAll(a.ApplicationID.String(), "-", ""), a.Environment)
}

func (a *Deployment) DatabaseMountVolume() string {
	return fmt.Sprintf("/var/sarabi/data/storage/%s-%s/data", strings.ReplaceAll(a.ApplicationID.String(), "-", ""), a.Environment)
}

func (a *Deployment) ContainerName(instanceId int) string {
	return fmt.Sprintf("%s-%s-%d", strings.ReplaceAll(a.ID.String(), "-", ""), a.Environment, instanceId)
}

func (a *Deployment) AccessURL(instanceType InstanceType) string {
	return fmt.Sprintf("%s-%s.%s", instanceType, a.Environment, a.Application.Domain)
}

func (a *Deployment) InternalAccessURL(instanceId int) string {
	return fmt.Sprintf("%s:%s", a.ContainerName(instanceId), a.Port)
}

func (a *Deployment) ProxyContainerName() string {
	return fmt.Sprintf("%s-%s", a.Application.Name, "proxy")
}

func (a *Deployment) SiteContentPath() string {
	return fmt.Sprintf("/var/caddy/share/%s", strings.ReplaceAll(a.ID.String(), "-", ""))
}

func (a *Deployment) BinPath() string {
	// /var/sarabi/data/bins/{application-uuid}/deployments/{deployment-uuid}
	return fmt.Sprintf("%s/bins/%s/deployments/%s.tar.gz", sarabiDataPath, a.ApplicationID, a.ID)
}
