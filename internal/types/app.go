package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"gorm.io/gorm"
	"io"
	"strings"
	"time"
)

var (
	sarabiDataPath         = "/var/sarabi/data"
	defaultAllocPercentage = 0.25
)

type (
	Application struct {
		ID             uuid.UUID            `gorm:"primaryKey"`
		Name           string               `json:"name"`
		Domain         string               `json:"domain"`
		StorageEngines StorageEngines       `json:"storage_engines"`
		Frontend       string               `json:"frontend"`
		Backend        string               `json:"backend"`
		Resources      ResourcesAllocations `json:"resources"`
		CreatedAt      time.Time            `json:"created_at"`
		UpdatedAt      time.Time            `json:"-"`
		DeletedAt      gorm.DeletedAt       `gorm:"index" json:"-"`
	}

	Deployment struct {
		ID            uuid.UUID    `gorm:"primaryKey" json:"id"`
		ApplicationID uuid.UUID    `gorm:"not null" json:"applicationID"`
		Environment   string       `json:"environment"`
		Status        string       `json:"status"`
		Instances     int          `json:"instances"`
		Name          string       `json:"name" gorm:"-"`
		Port          string       `json:"port"`
		InstanceType  InstanceType `json:"instance_type"`
		Identifier    string       `json:"identifier"`
		Application   Application  `gorm:"foreignKey:ApplicationID" json:"-"`
		CreatedAt     time.Time    `json:"created_at"`
	}

	NetworkAccess struct {
		ID            uuid.UUID   `gorm:"primaryKey" json:"id"`
		ApplicationID uuid.UUID   `gorm:"not null" json:"applicationID"`
		Application   Application `gorm:"foreignKey:ApplicationID" json:"-"`
		IP            string      `json:"ip"`
		Port          string      `json:"port"`
		Environment   string      `json:"environment"`
		CreatedAt     time.Time   `json:"created_at"`
		DeletedAt     time.Time   `json:"deleted_at"`
	}

	ResourceAllocation struct {
		CPU    int64 `json:"cpu"`
		Memory int64 `json:"memory"`
	}

	ResourceAllocationConfig struct {
		CPUPercentage    float64 `json:"cpu"`
		MemoryPercentage float64 `json:"memory"`
	}

	ResourcesAllocations map[StorageEngine]ResourceAllocationConfig

	DeploymentStatus string
	InstanceType     string
)

type (
	CreateApplicationParams struct {
		Name          string   `json:"name" binding:"required"`
		Domain        string   `json:"domain" binding:"required"`
		StorageEngine []string `json:"storage_engine" binding:"required"`
		Frontend      string   `json:"frontend"`
		Backend       string   `json:"backend"`
	}

	DeployParams struct {
		ApplicationID uuid.UUID
		Frontend      io.Reader
		Backend       io.Reader
		Instances     int
		Environment   string
		Identifier    string
	}

	CreateDeploymentParams struct {
		ApplicationID uuid.UUID `json:"application_id"`
		Environment   string    `json:"environment"`
		Instances     int
		Port          string       `json:"-"`
		InstanceType  InstanceType `json:"instance_type"` // frontend, backend, database, proxy
		Identifier    string       `json:"identifier"`
	}

	ContainerIdentity struct {
		DeploymentID uuid.UUID
		Environment  string
		InstanceID   int
		ID           string
	}
)

type StorageEngine string
type StorageEngines []StorageEngine

const (
	StorageEnginePostgres StorageEngine = "postgres"
	StorageEngineMysql    StorageEngine = "mysql"
	StorageEngineMongo    StorageEngine = "mongo"
	StorageEngineRedis    StorageEngine = "redis"
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

func (s StorageEngine) String() string {
	return string(s)
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

func (s ResourcesAllocations) Value() (driver.Value, error) {
	return json.Marshal(s)
}

func (s *ResourcesAllocations) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan []ResourcesAllocations: type assertion to []byte failed")
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

func (a *Deployment) SiteContentPath() string {
	return fmt.Sprintf("/var/caddy/share/%s", strings.ReplaceAll(a.ID.String(), "-", ""))
}

func (a *Deployment) BinPath() string {
	return fmt.Sprintf("%s/bins/%s/deployments/%s.tar.gz", sarabiDataPath, a.ApplicationID, a.ID)
}

func (a *Deployment) LogFilename() string {
	return fmt.Sprintf("%s-%s-%s.log", a.Application.Name, a.InstanceType, a.Environment)
}

func (a *Application) ResourcesAllocation(se StorageEngine) (ResourceAllocation, error) {
	defaultAlloc, err := hostResources()
	if err != nil {
		return ResourceAllocation{}, err
	}

	cfg, ok := a.Resources[se]
	if ok {
		return ResourceAllocation{
			CPU:    int64(float64(defaultAlloc.CPU) * cfg.CPUPercentage * 1e9),
			Memory: int64(float64(defaultAlloc.Memory) * cfg.MemoryPercentage),
		}, nil
	}

	return defaultStorageEngineResourceAllocation()
}

// defaultStorageEngineResourceAllocation returns the default resource allocation for a storage engine
// container. It allocates 25% of the total CPU and memory of the host machine.
func defaultStorageEngineResourceAllocation() (ResourceAllocation, error) {
	r, err := hostResources()
	if err != nil {
		return ResourceAllocation{}, err
	}

	return ResourceAllocation{
		CPU:    int64(float64(r.CPU) * defaultAllocPercentage * 1e9),
		Memory: int64(float64(r.Memory) * defaultAllocPercentage),
	}, nil
}

func hostResources() (ResourceAllocation, error) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return ResourceAllocation{}, err
	}

	cpuCount, err := cpu.Counts(true)
	if err != nil {
		return ResourceAllocation{}, err
	}

	return ResourceAllocation{
		CPU:    int64(cpuCount),
		Memory: int64(vm.Total),
	}, nil
}
