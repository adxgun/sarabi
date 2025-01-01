package logs

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/events"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"os"
	"sarabi"
	"sarabi/internal/database"
	"sarabi/internal/integrations/docker"
	"sarabi/internal/service"
	"sarabi/internal/storage"
	"sarabi/internal/types"
	"sarabi/logger"
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
		Read(ctx context.Context, applicationID uuid.UUID, filter types.Filter) (<-chan types.LogEntry, <-chan error)
		Register(applicationID uuid.UUID, environment string) <-chan LogEntry
	}

	manager struct {
		dockerClient       docker.Docker
		applicationService service.ApplicationService
		varsService        service.SecretService
		repo               database.LogsRepository

		clientsMu sync.Mutex
		watchers  map[string][]Client

		mu       sync.Mutex
		logFiles map[string]*os.File
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
	varsService service.SecretService) Manager {
	return &manager{
		dockerClient:       dc,
		applicationService: applicationService,
		varsService:        varsService,
		repo:               repo,
		watchers:           make(map[string][]Client),
		logFiles:           make(map[string]*os.File), mu: sync.Mutex{}}
}

func (m *manager) Watch(ctx context.Context) {
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
		logger.Info("ignoring event", zap.Any("action", ev.Action))
		return
	}

	containerName, ok := ev.Actor.Attributes["name"]
	if !ok {
		logger.Error("container name not found: ", zap.Any("event", ev.Actor))
		return
	}

	identity, err := sarabi.ParseContainerIdentity(ev.Actor.ID, containerName)
	if err != nil {
		logger.Error("error parsing container name",
			zap.Error(err),
			zap.String("container_name", containerName))
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
			logOwner := fmt.Sprintf("[%s-%d]", deployment.Application.Name, identity.InstanceID)
			scanner := bufio.NewScanner(logHandle)
			for scanner.Scan() {
				entry := LogEntry{Owner: logOwner, Log: scanner.Text()}
				val, err := json.Marshal(entry)
				if err != nil {
					logger.Error("failed to marshal log entry", zap.Error(err))
					continue
				}

				m.broadcast(entry, deployment.Application.ID, deployment.Environment)

				f, err := m.getLogFile(deployment)
				if err != nil {
					return errors.Wrap(err, "failed to get log file")
				}

				_, err = f.Write(val)
				if err != nil {
					return errors.Wrap(err, "failed to write log entry")
				}

				_, err = f.Write([]byte("\n"))
				if err != nil {
					return errors.Wrap(err, "failed to write new line")
				}

				if err := scanner.Err(); err != nil {
					logger.Warn("log scanner error", zap.Error(err))
					break
				}
			}
		}
	}
}

func (m *manager) stopStreaming(ctx context.Context, identity *types.ContainerIdentity) error {
	deployment, err := m.applicationService.GetDeployment(ctx, identity.DeploymentID)
	if err != nil {
		return errors.Wrap(err, "failed to fetch deployment")
	}

	logFile, err := os.Open(deployment.LogFilename())
	if err != nil {
		return errors.Wrap(err, "failed to get log file")
	}

	st, stType, err := m.getConfiguredStorage(ctx, deployment.ApplicationID)
	if err != nil {
		return errors.Wrap(err, "failed to get storage")
	}

	stat, err := logFile.Stat()
	if err != nil || stat == nil {
		return errors.Wrap(err, "failed to get log file stat")
	}

	logsLocation := fmt.Sprintf("%s/%s.log", storage.AppLogDir, uuid.New())
	if err := st.Save(ctx, logsLocation, types.File{
		Content: logFile,
		Stat:    types.FileStat{ContentType: "text/plain", Size: stat.Size()},
	}); err != nil {
		return errors.Wrap(err, "failed to save log file in storage")
	}

	entry := types.Log{
		ID:            uuid.New(),
		DeploymentID:  deployment.ID,
		ApplicationID: deployment.ApplicationID,
		Environment:   deployment.Environment,
		Location:      logsLocation,
		StorageType:   stType.String(),
		ContainerID:   identity.ID,
		Timestamp:     time.Now(),
	}
	if err := m.repo.Save(ctx, &entry); err != nil {
		return errors.Wrap(err, "failed to save log entry")
	}

	m.deleteLogFile(deployment)
	return nil
}

func (m *manager) getLogFile(dep *types.Deployment) (*os.File, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	logFileKey := dep.LogFilename()
	if f, ok := m.logFiles[logFileKey]; ok {
		return f, nil
	}

	// TODO: change file permission to read only by owner
	f, err := os.Create(logFileKey)
	if err != nil {
		return nil, err
	}

	m.logFiles[logFileKey] = f
	return f, nil
}

func (m *manager) Read(ctx context.Context, applicationID uuid.UUID, filter types.Filter) (<-chan types.LogEntry, <-chan error) {
	var ch = make(chan types.LogEntry)
	var errCh = make(chan error)

	logs, err := m.repo.FindAll(ctx, applicationID, filter)
	if err != nil {
		errCh <- err
		close(ch)
		close(errCh)
		return ch, errCh
	}

	go func(ctx context.Context, applicationID uuid.UUID) {
		for _, log := range logs {
			st, err := m.getStorage(ctx, applicationID, storage.Type(log.StorageType))
			if err != nil {
				errCh <- err
				return
			}

			file, err := st.Get(ctx, log.Location)
			if err != nil {
				errCh <- err
				return
			}

			scanner := bufio.NewScanner(file.Content)
			for scanner.Scan() {
				var entry types.LogEntry
				if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
					errCh <- err
					return
				}

				ch <- entry
			}

			if err := scanner.Err(); err != nil {
				errCh <- err
			}

			_ = file.Content.Close()
		}
	}(ctx, applicationID)

	return ch, errCh
}

func (m *manager) Register(applicationID uuid.UUID, environment string) <-chan LogEntry {
	ch := make(chan LogEntry)
	m.clientsMu.Lock()
	defer m.clientsMu.Unlock()

	key := fmt.Sprintf("%s-%s", applicationID, environment)
	m.watchers[key] = append(m.watchers[key], ch)
	return ch
}

func (m *manager) broadcast(entry LogEntry, applicationID uuid.UUID, environment string) {
	m.clientsMu.Lock()
	defer m.clientsMu.Unlock()

	key := fmt.Sprintf("%s-%s", applicationID, environment)
	for _, ch := range m.watchers[key] {
		ch <- entry
	}
}

func (m *manager) deleteLogFile(dep *types.Deployment) {
	// TODO: figure out what to do here
	m.mu.Lock()
	defer m.mu.Unlock()

	logFileKey := dep.LogFilename()
	if f, ok := m.logFiles[logFileKey]; ok {
		logger.Info("files before remove", zap.Any("files", m.logFiles))
		_ = f.Close()
		_ = os.Remove(logFileKey)
		delete(m.logFiles, logFileKey)
		logger.Info("files after remove", zap.Any("files", m.logFiles))
	}
}

func (m *manager) getConfiguredStorage(ctx context.Context, applicationID uuid.UUID) (storage.Storage, storage.Type, error) {
	creds, err := m.varsService.FindStorageCredentials(ctx, applicationID)
	if err != nil {
		logger.Info("cannot find object storage credentials, storing logs on filesystem", zap.Error(err))
		return storage.NewFileStorage(), storage.TypeFS, nil
	}

	st, err := storage.NewObjectStorage(*creds)
	if err != nil {
		logger.Info("cannot find object storage credentials, storing logs on filesystem", zap.Error(err))
		return storage.NewFileStorage(), storage.TypeFS, nil
	}
	return st, storage.TypeS3, nil
}

func (m *manager) getStorage(ctx context.Context, applicationID uuid.UUID, st storage.Type) (storage.Storage, error) {
	switch st {
	case storage.TypeFS:
		return storage.NewFileStorage(), nil
	case storage.TypeS3:
		creds, err := m.varsService.FindStorageCredentials(ctx, applicationID)
		if err != nil {
			return nil, err
		}
		return storage.NewObjectStorage(*creds)
	default:
		return nil, errors.New("unknown storage type")
	}
}
