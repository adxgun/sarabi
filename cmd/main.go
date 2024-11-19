package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sarabi"
	"sarabi/bundler"
	proxycomponent "sarabi/components/proxy"
	"sarabi/database"
	"sarabi/httphandlers"
	"sarabi/integrations/caddy"
	dockerclient "sarabi/integrations/docker"
	"sarabi/logger"
	"sarabi/manager"
	"sarabi/service"
	"syscall"
	"time"
)

func main() {
	if err := logger.InitLogger("development"); err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		return
	}
	defer logger.Sync()

	srv, err, teardown := setup()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		logger.Info("serving http(s) on :3646")
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal("server closed: ", err)
		}
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-done
	log.Println("Shutting down...")

	if teardown != nil {
		if err := teardown(); err != nil {
			logger.Error("teardown failed", zap.Error(err))
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %s\n", err)
	}
}

func setup() (*http.Server, error, func() error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	docker, err := dockerclient.NewDockerClient()
	if err != nil {
		return nil, err, nil
	}

	if err := docker.RunDind(ctx); err != nil {
		return nil, err, nil
	}

	db, err := database.Open(manager.DBDir)
	if err != nil {
		return nil, err, nil
	}

	deploymentRepo := database.NewDeploymentRepository(db)
	deploymentSecretRepo := database.NewDeploymentSecretRepository(db)
	appRepo := database.NewApplicationRepository(db)
	secretRepo := database.NewSecretRepository(db)

	encryptor := sarabi.NewEncryptor()
	appService := service.NewApplicationService(appRepo, deploymentRepo)
	secretService := service.NewSecretService(encryptor, secretRepo, deploymentSecretRepo)
	caddyClient := caddy.NewCaddyClient(appService)

	caddyProxy := proxycomponent.New(docker, appService, caddyClient)
	result, err := caddyProxy.Run(context.Background(), uuid.Nil)
	if err != nil {
		return nil, err, nil
	}

	mn := manager.New(appService, secretService, docker, caddyClient, bundler.NewArtifactStore())
	apiHandler := httphandlers.NewApiHandler(mn)
	routes := httphandlers.Routes(apiHandler)

	addr := ":3646"
	return &http.Server{
			Addr:    addr,
			Handler: routes,
		}, nil, func() error {
			return caddyProxy.Cleanup(context.Background(), result)
		}
}

/*
func _main() {
	if err := logger.InitLogger("development"); err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		return
	}
	defer logger.Sync()

	backendSourceDir := "/Users/lekanadigun/github/adigunhammedolalekan/fspasssample"
	// sourceDir := "/Users/lekanadigun/Documents/engine"
	buildFile := "example-app.tar.gz"
	dbDir := "fspaas.db"
	frontendSourceDir := "/Users/lekanadigun/Documents/frontend/dist"
	frontendBuildFile := "frontend-app.tar.gz"
	db, err := database.Open(dbDir)
	if err != nil {
		log.Fatal(err)
	}

	if err := sarabi.GzipDirectory(backendSourceDir, buildFile); err != nil {
		log.Fatal(err)
	}

	if err := sarabi.GzipDirectory(frontendSourceDir, frontendBuildFile); err != nil {
		log.Fatal(err)
	}

	docker, err := docker2.NewDockerClient()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("initialized Docker!")

	deploymentRepo := database.NewDeploymentRepository(db)
	deploymentSecretRepo := database.NewDeploymentSecretRepository(db)
	appRepo := database.NewApplicationRepository(db)
	secretRepo := database.NewSecretRepository(db)
	encryptor := sarabi.NewEncryptor()
	appService := service.NewApplicationService(appRepo, deploymentRepo)
	secretService := service.NewSecretService(encryptor, secretRepo, deploymentSecretRepo)
	caddyClient := caddy.NewCaddyClient(appService)
	ctx := context.Background()

	app, err := appService.Create(context.Background(), types.CreateApplicationParams{
		Name:   "frontend-test-0",
		Domain: "paas.local",
	})

	if err != nil {
		logger.Warn("failed to create app", zap.Error(err))
	}
	logger.Info("app created", zap.Any("app", app))

	deployment := &types.Deployment{
		ID:            uuid.New(),
		ApplicationID: app.ID,
		Environment:   "dev",
		Status:        "CREATED",
		BuildDir:      buildFile,
		Instances:     2,
		Application:   *app,
		Port:          "1995",
	}

	deploymentFrontend := &types.Deployment{
		ID:            uuid.New(),
		ApplicationID: app.ID,
		Environment:   "dev",
		Status:        "CREATED",
		BuildDir:      frontendBuildFile,
		Instances:     1,
		Application:   *app,
	}

	err = deploymentRepo.Save(ctx, deploymentFrontend)
	err = deploymentRepo.Save(ctx, deployment)
	if err != nil {
		log.Fatal("failed to create deployment")
	}

	logger.Info("deployment created", zap.Any("deployment", deployment))

	dbSecrets := []*types.Secret{
		{Name: "POSTGRES_DB", Value: app.Name, ID: uuid.New()},
		{Name: "POSTGRES_PASSWORD", Value: uuid.New().String(), ID: uuid.New()},
		{Name: "POSTGRES_USER", Value: app.Name + "-user", ID: uuid.New()},
		{Name: "DATABASE_HOST", Value: deployment.DBInstanceName(), ID: uuid.New()},
		{Name: "DATABASE_PORT", Value: "5432", ID: uuid.New()},
		{Name: "PORT", Value: deployment.Port, ID: uuid.New()},
	}

	createdSecrets := make([]*types.Secret, 0)
	for _, ss := range dbSecrets {
		v, err := secretService.Create(context.Background(), app.ID, ss.Name, ss.Value, "dev")
		if err != nil {
			logger.Error("failed to create secret", zap.Error(err))
		} else {
			logger.Info("secret created", zap.Any("secret", v))
			createdSecrets = append(createdSecrets, v)
		}
	}

	err = secretService.CreateDeploymentSecrets(ctx, deployment.ID, createdSecrets)
	if err != nil {
		log.Fatal("failed to create deployment secret", err)
	}

	fsComponents := []components.Builder{
		databasecomponent.New(docker, appService, secretService),
		proxycomponent.New(docker, appService, caddyClient),
		backendcomponent.New(docker, appService, secretService, caddyClient),
		frontendcomponent.New(docker, appService, secretService, caddyClient),
	}

	for _, fsC := range fsComponents {
		r, err := fsC.Run(ctx, deployment.ID)
		if err != nil {
			logger.Error("failed to start component",
				zap.Error(err),
				zap.String("name", fsC.Name()))
		} else {
			logger.Info("component started",
				zap.String("name", fsC.Name()),
				zap.Any("result", r))
		}

		if err := fsC.Cleanup(ctx, r); err != nil {
			logger.Error("cleanup failed",
				zap.Error(err))
		} else {
			logger.Info("cleanup completed!",
				zap.Any("result", r))
		}
	}
}
*/
