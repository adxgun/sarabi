package logs

import (
	"bufio"
	"context"
	"fmt"
	"github.com/docker/docker/api/types/events"
	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"math/rand"
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

type (
	Manager interface {
		Watch(ctx context.Context)
		Read(ctx context.Context, filter types.Filter) ([]types.LogEntry, error)
		ReadMem(filter types.Filter) ([]types.LogEntry, error)
	}

	manager struct {
		dockerClient       docker.Docker
		applicationService service.ApplicationService
		varsService        service.SecretService
		repo               database.LogsRepository
		lokiClient         loki.Client

		eb eventbus.Bus

		mu          sync.RWMutex
		entries     chan types.Batch
		batches     map[string][]types.Batch
		lastEntries *EvictingList[types.Batch]
	}
)

var (
	supportedActions = map[string]bool{
		"start":   true,
		"destroy": true,
	}

	identifierColors = map[string]*color.Color{}
	availableColors  = []*color.Color{
		color.New(color.FgHiRed),
		color.New(color.FgHiGreen),
		color.New(color.FgHiYellow),
		color.New(color.FgHiBlue),
		color.New(color.FgHiMagenta),
		color.New(color.FgHiCyan),
		color.New(color.FgRed),
		color.New(color.FgGreen),
		color.New(color.FgYellow),
		color.New(color.FgBlue),
		color.New(color.FgMagenta),
		color.New(color.FgCyan),
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
		entries:            make(chan types.Batch, 1000),
		batches:            make(map[string][]types.Batch),
		lastEntries:        NewEvictingList[types.Batch](50),
		eb:                 eb}
}

func (m *manager) Watch(ctx context.Context) {
	if err := m.restoreStreaming(ctx); err != nil {
		logger.Error("failed to restore streaming",
			zap.Error(err))
	}

	go m.aggregator(ctx)
	go m.flusher(ctx)

	evChan, errChan := m.dockerClient.ContainerEvents(context.Background())
	for {
		select {
		case ev := <-evChan:
			m.handleContainerEvent(ctx, ev)
		case err := <-errChan:
			if err != nil {
				logger.Error("container events error",
					zap.Error(err))
			}
			evChan, errChan = m.tryReconnect()
		case <-ctx.Done():
			logger.Info("stopping logs manager...")
			break
		}
	}
}

func (m *manager) tryReconnect() (<-chan events.Message, <-chan error) {
	time.Sleep(100 * time.Millisecond)
	return m.dockerClient.ContainerEvents(context.Background())
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
			logOwner := m.getColoredString(fmt.Sprintf("[%s-%s-%d]", deployment.Application.Name, deployment.Environment, identity.InstanceID))
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
			m.lastEntries.Add(batch)
		case <-ctx.Done():
			break
		}
	}
}

func (m *manager) flusher(ctx context.Context) {
	var (
		flushInterval = time.Second * 5
		ticker        = time.NewTicker(flushInterval)
	)

	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			batches := m.batches
			m.batches = make(map[string][]types.Batch)
			m.mu.Unlock()

			go m.sendBatches(ctx, batches)
		case <-ctx.Done():
			break
		}
	}
}

func (m *manager) sendBatches(ctx context.Context, batches map[string][]types.Batch) {
	// process 100 per batch
	for key, next := range batches {
		for i := 0; i < len(next); i += 100 {
			end := i + 100
			if end > len(next) {
				end = len(next)
			}

			next100Batch := make(map[string][]types.Batch)
			next100Batch[key] = next[i:end]
			err := m.lokiClient.Push(ctx, next100Batch)
			if err != nil {
				logger.Error("failed to push logs to loki.", zap.Error(err))
			}
		}
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

	logger.Info("restoring streaming for containers",
		zap.Any("containers", all))

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

func (m *manager) getColoredString(s string) string {
	if c, ok := identifierColors[s]; ok {
		return c.Sprintf("%s", s)
	}

	c := availableColors[rand.Intn(len(availableColors))]
	identifierColors[s] = c
	return c.Sprintf("%s", s)
}

func (m *manager) ReadMem(filter types.Filter) ([]types.LogEntry, error) {
	response := make([]types.LogEntry, 0)
	for _, next := range m.lastEntries.Values() {
		key := fmt.Sprintf("%s-%s", next.Deployment.ApplicationID, next.Deployment.Environment)
		if filter.Identifier == key {
			response = append(response, next.Log)
		}
	}
	return response, nil
}
