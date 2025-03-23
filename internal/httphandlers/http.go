package httphandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"net/http"
	"sarabi/internal/eventbus"
	"sarabi/internal/logs"
	"sarabi/internal/manager"
	"sarabi/internal/misc"
	"sarabi/internal/types"
	"sarabi/logger"
	"strconv"
	"strings"
	"time"
)

var (
	maxUploadSize = 2 << 30 // 2GB
)

type (
	ApiHandler struct {
		mn     manager.Manager
		lm     logs.Manager
		eb     eventbus.Bus
		logger *zap.Logger
	}
)

func NewApiHandler(mn manager.Manager, lm logs.Manager, eb eventbus.Bus, l *zap.Logger) *ApiHandler {
	return &ApiHandler{mn: mn, lm: lm, eb: eb, logger: l}
}

func (handler *ApiHandler) CreateApplication(w http.ResponseWriter, r *http.Request) {
	var params types.CreateApplicationParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		badRequest(w, err)
		return
	}

	app, err := handler.mn.CreateApplication(r.Context(), params)
	if err != nil {
		serverError(w, err)
		return
	}

	ok(w, "application created", app)
}

func (handler *ApiHandler) Deploy(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		badRequest(w, errors.Wrap(err, "failed to parse uploads"))
		return
	}

	var body struct {
		ApplicationID uuid.UUID `json:"application_id"`
		Instances     int       `json:"instances"`
		Environment   string    `json:"environment"`
	}

	if err := json.Unmarshal([]byte(r.FormValue("json")), &body); err != nil {
		badRequest(w, errors.Wrap(err, "invalid request body"))
		return
	}

	identifier, err := misc.DefaultRandomIdGenerator.Generate(10)
	if err != nil {
		serverError(w, err)
		return
	}

	param := &types.DeployParams{
		ApplicationID: body.ApplicationID,
		Instances:     body.Instances,
		Environment:   body.Environment,
		Identifier:    identifier,
	}
	for _, ff := range r.MultipartForm.File["files"] {
		if !strings.HasSuffix(ff.Filename, ".tar.gz") {
			badRequest(w, errors.New("unknown upload type: "+ff.Filename))
			return
		}

		if ff.Size > int64(maxUploadSize) {
			badRequest(w, errors.New("upload too large: "+ff.Filename))
			return
		}

		file, err := ff.Open()
		if err != nil {
			logger.Error("failed to process deployment upload: ",
				zap.Error(err))
			badRequest(w, errors.New("failed to open file upload"))
			return
		}

		if strings.Contains(ff.Filename, "frontend") {
			param.Frontend = file
		}

		if strings.Contains(ff.Filename, "backend") {
			param.Backend = file
		}
	}

	ch := handler.eb.Register(identifier)

	logger.Info("starting deployment",
		zap.Any("application_id", param.ApplicationID))
	go func(ctx context.Context) {
		err = handler.mn.Deploy(ctx, param)
		if err != nil {
			ch <- eventbus.Event{
				Type:    eventbus.Error,
				Message: err.Error(),
			}
		}
	}(r.Context())

	for {
		select {
		case ev := <-ch:
			data, err := json.Marshal(ev)
			if err != nil {
				serverError(w, err)
				continue
			}

			_, _ = w.Write(data)
			_, _ = w.Write(misc.Seperator)
			flusher, ok := w.(http.Flusher)
			if ok {
				flusher.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

func (handler *ApiHandler) UpdateVariables(w http.ResponseWriter, r *http.Request) {
	applicationID, err := uuid.Parse(chi.URLParam(r, "application_id"))
	if err != nil {
		badRequest(w, err)
		return
	}

	var body struct {
		Environment string                     `json:"environment"`
		Secrets     []types.CreateSecretParams `json:"vars"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err = handler.mn.UpdateVariables(ctx, applicationID, body.Environment, body.Secrets...)
	if err != nil {
		serverError(w, err)
		return
	}

	ok(w, "variable updated", nil)
}

func (handler *ApiHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Identifier string `json:"identifier"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, err)
		return
	}

	if len(body.Identifier) < 10 {
		badRequest(w, errors.New("invalid deployment identifier"))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := handler.mn.Rollback(ctx, body.Identifier)
	if err != nil {
		serverError(w, err)
		return
	}

	ok(w, "rollback completed", result)
}

func (handler *ApiHandler) Scale(w http.ResponseWriter, r *http.Request) {
	applicationID, err := uuid.Parse(chi.URLParam(r, "application_id"))
	if err != nil {
		badRequest(w, err)
		return
	}

	var body struct {
		Count       int    `json:"count"`
		Environment string `json:"environment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, err)
		return
	}

	if body.Count <= 0 {
		badRequest(w, fmt.Errorf("invalid instance count: %d", body.Count))
		return
	}

	if len(body.Environment) == 0 {
		badRequest(w, fmt.Errorf("invalid environment value: %s", body.Environment))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := handler.mn.Scale(ctx, applicationID, body.Environment, body.Count)
	if err != nil {
		serverError(w, err)
		return
	}

	ok(w, "deployment changed", result)
}

func (handler *ApiHandler) AddDomain(w http.ResponseWriter, r *http.Request) {
	applicationID, err := uuid.Parse(chi.URLParam(r, "application_id"))
	if err != nil {
		badRequest(w, err)
		return
	}

	params := &types.AddDomainParams{}
	if err := json.NewDecoder(r.Body).Decode(params); err != nil {
		badRequest(w, err)
		return
	}

	domain, err := handler.mn.AddDomain(r.Context(), applicationID, *params)
	if err != nil {
		serverError(w, err)
		return
	}

	ok(w, "domain added", domain)
}

func (handler *ApiHandler) RemoveDomain(w http.ResponseWriter, r *http.Request) {
	applicationID, err := uuid.Parse(chi.URLParam(r, "application_id"))
	if err != nil {
		badRequest(w, err)
		return
	}

	var body struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, err)
		return
	}

	err = handler.mn.RemoveDomain(r.Context(), applicationID, body.Name)
	if err != nil {
		serverError(w, err)
		return
	}

	ok(w, "domain removed", nil)
}

func (handler *ApiHandler) AddCredentials(w http.ResponseWriter, r *http.Request) {
	params := types.AddCredentialsParams{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		badRequest(w, err)
		return
	}

	result, err := handler.mn.AddCredentials(r.Context(), params)
	if err != nil {
		serverError(w, err)
		return
	}

	ok(w, "credentials added", result)
}

func (handler *ApiHandler) DownloadBackup(w http.ResponseWriter, r *http.Request) {
	backupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		badRequest(w, err)
		return
	}

	result, err := handler.mn.DownloadBackup(r.Context(), backupID)
	if err != nil {
		serverError(w, err)
		return
	}

	w.Header().Add("Content-Length", fmt.Sprintf("%d", result.Stat.Size))
	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s", result.Stat.Name))
	w.WriteHeader(http.StatusOK)
	buf := make([]byte, 10*1024*1024) // 10MB
	for {
		n, err := result.Content.Read(buf)
		if err != nil && err != io.EOF {
			serverError(w, err)
			break
		}

		if n > 0 {
			if _, err = w.Write(buf[:n]); err != nil {
				serverError(w, err)
				break
			}
		}

		if err == io.EOF {
			break
		}
	}
}

func (handler *ApiHandler) ListBackups(w http.ResponseWriter, r *http.Request) {
	applicationID, err := uuid.Parse(chi.URLParam(r, "application_id"))
	if err != nil {
		badRequest(w, err)
		return
	}

	environment := r.URL.Query().Get("environment")
	result, err := handler.mn.ListBackups(r.Context(), applicationID, environment)
	if err != nil {
		badRequest(w, err)
		return
	}

	ok(w, "success", result)
}

func (handler *ApiHandler) ListApplications(w http.ResponseWriter, r *http.Request) {
	apps, err := handler.mn.ListApplications(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}

	ok(w, "success", apps)
}

func (handler *ApiHandler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	applicationID, err := uuid.Parse(chi.URLParam(r, "application_id"))
	if err != nil {
		badRequest(w, err)
		return
	}

	deps, err := handler.mn.ListDeployments(r.Context(), applicationID)
	if err != nil {
		serverError(w, err)
		return
	}

	ok(w, "success", deps)
}

func (handler *ApiHandler) Destroy(w http.ResponseWriter, r *http.Request) {
	applicationID, err := uuid.Parse(chi.URLParam(r, "application_id"))
	if err != nil {
		badRequest(w, err)
		return
	}

	var body struct {
		Environment string `json:"environment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err = handler.mn.Destroy(ctx, applicationID, body.Environment)
	if err != nil {
		logger.Error("destroy failed",
			zap.Error(err))
		serverError(w, err)
		return
	}

	ok(w, "success", nil)
}

func (handler *ApiHandler) ListVariables(w http.ResponseWriter, r *http.Request) {
	applicationID, err := uuid.Parse(chi.URLParam(r, "application_id"))
	if err != nil {
		badRequest(w, err)
		return
	}

	environment := r.URL.Query().Get("environment")
	values, err := handler.mn.ListVariables(r.Context(), applicationID, &environment)
	if err != nil {
		serverError(w, err)
		return
	}

	ok(w, "success", values)
}

func (handler *ApiHandler) GetApplication(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	var (
		application *types.Application
		err         error
	)

	if id := queryParams.Get("id"); id != "" {
		applicationID, err := uuid.Parse(id)
		if err != nil {
			badRequest(w, err)
			return
		}
		application, err = handler.mn.GetApplication(r.Context(), &applicationID, nil)
		if err != nil {
			serverError(w, err)
			return
		}
	}

	if name := queryParams.Get("name"); name != "" {
		application, err = handler.mn.GetApplication(r.Context(), nil, &name)
		if err != nil {
			serverError(w, err)
			return
		}
	}

	ok(w, "success", application)
}

func (handler *ApiHandler) WhitelistIP(w http.ResponseWriter, r *http.Request) {
	applicationID, err := uuid.Parse(chi.URLParam(r, "application_id"))
	if err != nil {
		badRequest(w, err)
		return
	}

	var body struct {
		IP          string `json:"ip"`
		Environment string `json:"environment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, err)
		return
	}

	err = handler.mn.ManageDatabaseNetworkAccess(r.Context(), applicationID, body.Environment, body.IP, manager.OpAdd)
	if err != nil {
		serverError(w, err)
		return
	}

	ok(w, "IP whitelisted!", nil)
}

func (handler *ApiHandler) BlacklistIP(w http.ResponseWriter, r *http.Request) {
	applicationID, err := uuid.Parse(chi.URLParam(r, "application_id"))
	if err != nil {
		badRequest(w, err)
		return
	}

	var body struct {
		IP          string `json:"ip"`
		Environment string `json:"environment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, err)
		return
	}

	err = handler.mn.ManageDatabaseNetworkAccess(r.Context(), applicationID, body.Environment, body.IP, manager.OpRemove)
	if err != nil {
		serverError(w, err)
		return
	}

	ok(w, "IP blacklisted!", nil)
}

func (handler *ApiHandler) CreateBackup(w http.ResponseWriter, r *http.Request) {
	applicationID, err := uuid.Parse(chi.URLParam(r, "application_id"))
	if err != nil {
		badRequest(w, err)
		return
	}

	var body struct {
		CronExpression string `json:"cron_expression"`
		Environment    string `json:"environment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, err)
		return
	}

	err = handler.mn.CreateBackupSchedule(r.Context(), applicationID, body.Environment, body.CronExpression)
	if err != nil {
		serverError(w, err)
		return
	}

	ok(w, "backup created", nil)
}

func (handler *ApiHandler) StreamLogs(w http.ResponseWriter, r *http.Request) {
	applicationID, err := uuid.Parse(chi.URLParam(r, "application_id"))
	if err != nil {
		badRequest(w, err)
		return
	}

	queries := r.URL.Query()
	environment := queries.Get("environment")
	since := queries.Get("since")
	startAt := queries.Get("start")
	endAt := queries.Get("end")
	nLimit, _ := strconv.Atoi(queries.Get("limit"))

	limit := int64(nLimit)
	filterParams := types.FilterParams{
		Environment: environment,
		Start:       &startAt,
		End:         &endAt,
		Since:       &since,
		Limit:       &limit,
	}

	filter, err := filterParams.Validate()
	if err != nil {
		badRequest(w, err)
		return
	}

	filter.ApplicationID = applicationID
	entries, err := handler.lm.Read(r.Context(), *filter)
	if err != nil {
		serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for _, e := range entries {
		_ = writeSSELine(w, eventbus.Event{Type: eventbus.Info, Message: e.Log})
	}

	_ = writeSSELine(w, eventbus.Event{Type: eventbus.Complete})
}

func (handler *ApiHandler) TailLogs(w http.ResponseWriter, r *http.Request) {
	applicationID, err := uuid.Parse(chi.URLParam(r, "application_id"))
	if err != nil {
		badRequest(w, err)
		return
	}

	environment := r.URL.Query().Get("environment")
	if environment == "" {
		badRequest(w, errors.New("environment is required"))
		return
	}

	limit := r.URL.Query().Get("limit")
	var nLimit int
	if limit != "" {
		nLimit, err = strconv.Atoi(limit)
		if err != nil {
			badRequest(w, err)
			return
		}
	}

	if nLimit == 0 {
		nLimit = 30
	}

	identifier := fmt.Sprintf("%s-%s", applicationID, environment)
	lg := handler.logger.With(
		zap.String("identifier", identifier),
		zap.String("environment", environment),
		zap.Any("application_id", applicationID),
		zap.Int("limit", nLimit))

	filter := types.Filter{
		Environment:   environment,
		Since:         "5m",
		ApplicationID: applicationID,
		Identifier:    identifier,
		Limit:         int64(nLimit),
	}

	entries, err := handler.lm.ReadMem(filter)
	if err != nil {
		serverError(w, err)
		return
	}

	for _, e := range entries {
		_ = writeSSELine(w, eventbus.Event{Type: eventbus.Info, Message: e.Log})
	}

	ch := handler.eb.Register(identifier)
	lg.Info("registered client for log stream")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		select {
		case logEntry, ok := <-ch:
			if !ok {
				serverError(w, errors.New("log stream closed"))
				break
			}

			_ = writeSSELine(w, logEntry)
		case <-r.Context().Done():
			logger.Info("client disconnected")
			return
		}
	}
}

func (handler *ApiHandler) Ping(w http.ResponseWriter, r *http.Request) {
	err := handler.mn.Ping(r.Context(), r.Header.Get(authorizationHeader))
	if err != nil {
		unauthorized(w, err)
		return
	}

	ok(w, "success", nil)
}
