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
	"sarabi/internal/bundler"
	proxycomponent "sarabi/internal/components/proxy"
	"sarabi/internal/database"
	"sarabi/internal/eventbus"
	"sarabi/internal/firewall"
	"sarabi/internal/httphandlers"
	"sarabi/internal/integrations/caddy"
	dockerclient "sarabi/internal/integrations/docker"
	"sarabi/internal/logs"
	"sarabi/internal/manager"
	"sarabi/internal/misc"
	"sarabi/internal/service"
	"sarabi/internal/storage"
	"sarabi/logger"
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
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

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
	eventBus := eventbus.New()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	docker, err := dockerclient.NewClient(eventBus)
	if err != nil {
		return nil, err, nil
	}

	db, err := database.Open(storage.DBDir)
	if err != nil {
		return nil, err, nil
	}

	deploymentRepo := database.NewDeploymentRepository(db)
	deploymentSecretRepo := database.NewDeploymentSecretRepository(db)
	appRepo := database.NewApplicationRepository(db)
	secretRepo := database.NewSecretRepository(db)
	domainRepo := database.NewDomainRepository(db)
	backupSettingsRepo := database.NewBackupSettingsRepository(db)
	credentialRepo := database.NewServerConfigRepository(db)
	backupRepository := database.NewBackupRepository(db)
	naRepository := database.NewNetworkAccessRepository(db)
	logsRepository := database.NewLogsRepository(db)

	encryptor := misc.NewEncryptor()
	appService := service.NewApplicationService(appRepo, deploymentRepo)
	secretService := service.NewSecretService(encryptor, secretRepo, deploymentSecretRepo, credentialRepo)
	caddyClient := caddy.NewClient(eventBus)
	domainService := service.NewDomainService(caddyClient, domainRepo)
	fm := firewall.NewManager()
	logsManager := logs.NewManager(docker, appService, logsRepository, secretService)

	backupSvc, err := service.NewBackupService(docker, appService, secretService, backupSettingsRepo, backupRepository)
	if err != nil {
		return nil, err, nil
	}

	if err := backupSvc.Run(ctx); err != nil {
		return nil, err, nil
	}

	caddyProxy := proxycomponent.New(docker, appService, caddyClient)
	_, err = caddyProxy.Run(ctx, uuid.Nil)
	if err != nil {
		return nil, err, nil
	}

	go func() {
		logsManager.Watch(ctx)
	}()

	mn := manager.New(appService, secretService, docker, caddyClient,
		bundler.NewArtifactStore(), domainService, backupSvc, fm, naRepository, eventBus)
	apiHandler := httphandlers.NewApiHandler(mn, logsManager, eventBus)
	routes := httphandlers.Routes(apiHandler)

	addr := ":3646"
	return &http.Server{
			Addr:    addr,
			Handler: routes,
		}, nil, func() error {
			sqlDB, _ := db.DB()
			if sqlDB != nil {
				err = sqlDB.Close()
				logger.Info("DB Closed", zap.Error(err))
			}

			cancel()
			// return caddyProxy.Cleanup(context.Background(), result)
			return nil
		}
}
