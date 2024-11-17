package databasecomponent

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"sarabi/components"
	"sarabi/integrations/docker"
	"sarabi/logger"
	"sarabi/service"
)

const (
	databaseImageName = "postgres:17"
	databasePort      = 5432
	databasePath      = "/var/lib/postgresql/data"
)

type databaseComponent struct {
	dockerClient  docker.Docker
	appService    service.ApplicationService
	secretService service.SecretService
}

func New(dc docker.Docker, appSvc service.ApplicationService, secretService service.SecretService) components.Builder {
	return &databaseComponent{
		dockerClient:  dc,
		appService:    appSvc,
		secretService: secretService,
	}
}

func (d *databaseComponent) Name() string {
	return "database:" + databaseImageName
}

func (d *databaseComponent) Run(ctx context.Context, deploymentID uuid.UUID) (*components.BuilderResult, error) {
	deployment, err := d.appService.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, err
	}

	running, info := d.dockerClient.IsContainerRunning(ctx, deployment.DBInstanceName())
	if running {
		return &components.BuilderResult{ID: info.ID, Name: info.Name}, nil
	}

	dbVars, err := d.secretService.FindDeploymentSecrets(ctx, deploymentID)
	if err != nil {
		return nil, err
	}

	if err := d.dockerClient.CreateNetwork(ctx, deployment.NetworkName()); err != nil {
		return nil, err
	}

	if err := d.dockerClient.PullImage(ctx, databaseImageName); err != nil {
		return nil, err
	}

	// TODO:
	// 1. write postgresql config type
	//  PostgresSQLConfig struct {
	//    MaxConnections      int // 300
	//    SharedBuffers       string // 30% of total memory
	//    WorkMem             string // 16MB
	//    MaintenanceWorkMem  string // 200MB
	//}

	var envs []string
	for _, ss := range dbVars {
		envs = append(envs, ss.Env())
	}
	logger.Info("db envs", zap.Any("envs", envs))
	volumeMounts := []string{fmt.Sprintf("%s:%s", deployment.DatabaseMountVolume(), databasePath)}
	/*port, _ := nat.NewPort("tcp", "5432")
	hostBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: "5432",
	}
	portBinding := nat.PortMap{port: []nat.PortBinding{hostBinding}}*/
	startResp, err := d.dockerClient.StartContainerAndWait(ctx,
		databaseImageName, deployment.DBInstanceName(), deployment.NetworkName(), volumeMounts, envs, nil, nil)
	if err != nil {
		return nil, err
	}

	return &components.BuilderResult{
		ID:   startResp.ID,
		Name: startResp.Name,
	}, nil
}

func (d *databaseComponent) Cleanup(ctx context.Context, result *components.BuilderResult) error {
	return nil
}
