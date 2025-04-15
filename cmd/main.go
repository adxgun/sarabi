package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sarabi/internal/bundler"
	"sarabi/internal/components/logcollector"
	proxycomponent "sarabi/internal/components/proxy"
	"sarabi/internal/config"
	"sarabi/internal/database"
	"sarabi/internal/eventbus"
	"sarabi/internal/firewall"
	"sarabi/internal/httphandlers"
	"sarabi/internal/integrations/caddy"
	dockerclient "sarabi/internal/integrations/docker"
	"sarabi/internal/integrations/loki"
	"sarabi/internal/logs"
	"sarabi/internal/manager"
	"sarabi/internal/misc"
	"sarabi/internal/service"
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

	cfg := config.New()
	srv, err, teardown := setup(cfg)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		logger.Info(context.Background(), "serving http(s) on :"+cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server closed: ", err)
		}
	}()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	<-done
	log.Println("Shutting down...")

	if teardown != nil {
		if err := teardown(); err != nil {
			logger.Error(context.Background(), "teardown failed", zap.Error(err))
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %s\n", err)
	}
}

func setup(cfg config.Config) (*http.Server, error, func() error) {
	eventBus := eventbus.New()
	lokiClient := loki.NewClient()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	docker, err := dockerclient.NewClient(eventBus)
	if err != nil {
		return nil, err, nil
	}

	db, err := database.Open(cfg.DatabasePath)
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

	encryptor := misc.NewEncryptor(cfg.EncryptionKey)
	appService := service.NewApplicationService(appRepo, deploymentRepo)
	secretService := service.NewSecretService(encryptor, secretRepo, deploymentSecretRepo, credentialRepo)
	domainService := service.NewDomainService(domainRepo)
	caddyClient := caddy.NewClient(eventBus, domainService, cfg)

	logCollector := logcollector.New(docker, lokiClient, secretService)
	if _, err := logCollector.Run(ctx, uuid.Nil); err != nil {
		return nil, err, nil
	}

	fm := firewall.NewManager()
	logsManager := logs.NewManager(docker, appService, logsRepository, secretService, lokiClient, eventBus)

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

	// TODO: init caddy based on its saved state
	if err := caddyClient.Init(ctx); err != nil {
		return nil, errors.Wrap(err, "caddy failed to init"), nil
	}

	if cfg.Domain != "" {
		_, err = url.Parse(cfg.Domain)
		if err != nil {
			return nil, fmt.Errorf("invalid domain URL: %w", err), nil
		}
		if err := caddyClient.SetupSarabiAccess(ctx, cfg.Domain); err != nil {
			return nil, err, nil
		}
	}

	mn := manager.New(appService, secretService, docker, caddyClient,
		bundler.NewArtifactStore(), domainService, backupSvc, fm, naRepository, eventBus, cfg)
	apiHandler := httphandlers.NewApiHandler(mn, logsManager, eventBus, logger.GetLogger())
	routes := httphandlers.Routes(apiHandler)

	addr := fmt.Sprintf(":%s", cfg.Port)
	return &http.Server{
			Addr:    addr,
			Handler: routes,
		}, nil, func() error {
			sqlDB, _ := db.DB()
			if sqlDB != nil {
				if err = sqlDB.Close(); err != nil {
					logger.Info(context.Background(), "DB closed with error", zap.Error(err))
				}
			}
			cancel()
			// return caddyProxy.Cleanup(context.Background(), result)
			return nil
		}
}
