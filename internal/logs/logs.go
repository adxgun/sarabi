package logs

import (
	"bufio"
	"context"
	"fmt"
	"github.com/docker/docker/api/types/events"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"sarabi/internal/database"
	"sarabi/internal/eventbus"
	"sarabi/internal/integrations/docker"
	"sarabi/internal/integrations/loki"
	"sarabi/internal/misc"
	"sarabi/internal/service"
	"sarabi/internal/types"
	"sarabi/logger"
	"strconv"
	"strings"
	"sync"
	"time"
)

// how it works
// 1. Manager.Run() starts a goroutine that listens to container events
// 2. When a container starts, it starts streaming logs from the container, saving them to a temp file
// 3. When a container is destroyed, it stops streaming logs, saves the logs to a storage, and deletes the temp file
// 4. Manager.Read() reads logs from the storage and returns them

type (
	Manager interface {
		Watch(ctx context.Context)
		Read(ctx context.Context, filter types.Filter) ([]types.LogEntry, error)
	}

	manager struct {
		dockerClient       docker.Docker
		applicationService service.ApplicationService
		varsService        service.SecretService
		repo               database.LogsRepository
		lokiClient         loki.Client

		eb eventbus.Bus

		mu       sync.RWMutex
		entries  chan types.Batch
		batches  map[string][]types.Batch
		backlogs map[string][]types.Batch
	}
)

var (
	supportedActions = map[string]bool{
		"start":   true,
		"destroy": true,
	}
)

func NewManager(
	dc docker.Docker,
	applicationService service.ApplicationService,
	repo database.LogsRepository,
	varsService service.SecretService,
	lokiClient loki.Client,
	eb eventbus.Bus) Manager {
	return &manager{
		dockerClient:       dc,
		applicationService: applicationService,
		varsService:        varsService,
		repo:               repo,
		lokiClient:         lokiClient,
		backlogs:           make(map[string][]types.Batch),
		entries:            make(chan types.Batch, 1000),
		batches:            make(map[string][]types.Batch),
		eb:                 eb}
}

func (m *manager) Watch(ctx context.Context) {
	if err := m.restoreStreaming(ctx); err != nil {
		logger.Error("failed to restore streaming",
			zap.Error(err))
	}

	go m.aggregator(ctx)
	go m.flusher(ctx)

	evChan, errChan := m.dockerClient.ContainerEvents(ctx)
	for {
		select {
		case ev := <-evChan:
			m.handleContainerEvent(ctx, ev)
		case err := <-errChan:
			if err != nil {
				logger.Error("container events error",
					zap.Error(err))
			}
			evChan, errChan = m.tryReconnect(ctx)
		case <-ctx.Done():
			logger.Info("stopping logs manager...")
			break
		}
	}
}

func (m *manager) tryReconnect(ctx context.Context) (<-chan events.Message, <-chan error) {
	time.Sleep(100 * time.Millisecond)
	return m.dockerClient.ContainerEvents(ctx)
}

func (m *manager) handleContainerEvent(ctx context.Context, ev events.Message) {
	if !supportedActions[string(ev.Action)] {
		return
	}

	name, ok := ev.Actor.Attributes["name"]
	if !ok {
		return
	}

	identity, err := misc.ParseContainerIdentity(ev.Actor.ID, name)
	if err != nil {
		return
	}

	switch containerEvent(ev.Action) {
	case containerStart:
		go func(ctx context.Context, id *types.ContainerIdentity) {
			if err := m.startStreaming(ctx, id); err != nil {
				logger.Error("startStreaming: container log streaming error",
					zap.Error(err))
			}
		}(ctx, identity)
	case containerDestroy:
		go func(ctx context.Context, id *types.ContainerIdentity) {
			if err := m.stopStreaming(ctx, id); err != nil {
				logger.Error("stopStreaming: container log streaming error",
					zap.Error(err))
			}
		}(ctx, identity)
	}
}

func (m *manager) startStreaming(ctx context.Context, identity *types.ContainerIdentity) error {
	logger.Info("starting log streaming",
		zap.Any("identity", identity))
	deployment, err := m.applicationService.GetDeployment(ctx, identity.DeploymentID)
	if err != nil {
		return errors.Wrap(err, "failed to fetch deployment")
	}

	logHandle, err := m.dockerClient.ContainerLogs(ctx, identity.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get container logs")
	}

	defer func() {
		_ = logHandle.Close()
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			logOwner := fmt.Sprintf("[%s-%s-%d]", deployment.Application.Name, deployment.Environment, identity.InstanceID)
			scanner := bufio.NewScanner(logHandle)
			for scanner.Scan() {
				entry := types.LogEntry{Owner: logOwner, Log: scanner.Text(), Ts: strconv.FormatInt(time.Now().UnixNano(), 10)}
				m.entries <- types.Batch{
					Log:        entry,
					Deployment: deployment,
				}

				m.broadcast(entry, deployment.Application.ID, deployment.Environment)

				if err := scanner.Err(); err != nil {
					logger.Warn("log scanner error", zap.Error(err))
					break
				}
			}
		}
	}
}

func (m *manager) stopStreaming(ctx context.Context, identity *types.ContainerIdentity) error {
	_, err := m.applicationService.GetDeployment(ctx, identity.DeploymentID)
	if err != nil {
		return errors.Wrap(err, "failed to fetch deployment")
	}

	return nil
}

func (m *manager) aggregator(ctx context.Context) {
	for {
		select {
		case batch := <-m.entries:
			m.mu.Lock()
			m.batches[batch.Deployment.ID.String()] = append(m.batches[batch.Deployment.ID.String()], batch)
			m.mu.Unlock()
		case <-ctx.Done():
			break
		}
	}
}

func (m *manager) flusher(ctx context.Context) {
	var (
		flushInterval        = time.Second * 10
		backlogFlushInterval = time.Second * 30
		ticker               = time.NewTicker(flushInterval)
		backlogTicker        = time.NewTicker(backlogFlushInterval)
	)

	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			batches := m.batches
			m.batches = make(map[string][]types.Batch)
			m.mu.Unlock()

			go m.sendBatches(ctx, batches)
		case <-backlogTicker.C:
			m.mu.Lock()
			batches := m.backlogs
			m.backlogs = make(map[string][]types.Batch)
			m.mu.Unlock()
			go m.sendBatches(ctx, batches)
		case <-ctx.Done():
			break
		}
	}
}

func (m *manager) sendBatches(ctx context.Context, batches map[string][]types.Batch) {
	err := m.lokiClient.Push(ctx, batches)
	if err != nil {
		logger.Error("failed to push logs to loki. adding to backlogs...", zap.Error(err))
		m.mu.Lock()
		for id, values := range batches {
			m.backlogs[id] = append(m.backlogs[id], values...)
		}
		m.mu.Unlock()
	}
}

func (m *manager) Read(ctx context.Context, filter types.Filter) ([]types.LogEntry, error) {
	return m.lokiClient.Query(ctx, filter)
}

func (m *manager) broadcast(entry types.LogEntry, applicationID uuid.UUID, environment string) {
	key := fmt.Sprintf("%s-%s", applicationID, environment)
	m.eb.Broadcast(key, eventbus.Info, entry.Line())
}

func (m *manager) restoreStreaming(ctx context.Context) error {
	all, err := m.dockerClient.ListContainers(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to list containers")
	}

	logger.Info("restoring streaming for containers", zap.Any("containers", all))

	for _, c := range all {
		if c.State == "running" {
			identity, err := misc.ParseContainerIdentity(c.ID, strings.Replace(c.Name, "/", "", 1))
			if err != nil {
				continue
			}

			go func(ctx context.Context, id *types.ContainerIdentity) {
				if err := m.startStreaming(ctx, id); err != nil {
					logger.Error("startStreaming: container log streaming error",
						zap.Error(err))
				}
			}(ctx, identity)
		}
	}

	return nil
}
