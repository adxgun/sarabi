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
	"sarabi/internal/manager"
	"sarabi/internal/types"
	"sarabi/logger"
	"strings"
	"time"
)

var (
	maxUploadSize = 2 << 30 // 2GB
)

type (
	ApiHandler struct {
		mn manager.Manager
	}
)

func NewApiHandler(mn manager.Manager) *ApiHandler {
	return &ApiHandler{mn: mn}
}

func (handler *ApiHandler) CreateApplication(w http.ResponseWriter, r *http.Request) {
	var params types.CreateApplicationParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		badRequest(w, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app, err := handler.mn.CreateApplication(ctx, params)
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

	param := &types.DeployParams{
		ApplicationID: body.ApplicationID,
		Instances:     body.Instances,
		Environment:   body.Environment,
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

	logger.Info("starting deployment",
		zap.Any("application_id", param.ApplicationID))
	resp, err := handler.mn.Deploy(context.Background(), param)
	if err != nil {
		serverError(w, errors.Wrap(err, "deployment failed"))
		return
	}

	ok(w, "deployment succeeded", resp)
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

	err = handler.mn.UpdateVariables(context.Background(), applicationID, body.Environment, body.Secrets...)
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

	result, err := handler.mn.Rollback(context.Background(), body.Identifier)
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
		Count int `json:"count"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		badRequest(w, err)
		return
	}

	if body.Count <= 0 {
		badRequest(w, fmt.Errorf("invalid instance count: %d", body.Count))
		return
	}

	result, err := handler.mn.Scale(context.Background(), applicationID, body.Count)
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

	domain, err := handler.mn.AddDomain(context.Background(), applicationID, *params)
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

	err = handler.mn.RemoveDomain(context.Background(), applicationID, body.Name)
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

	result, err := handler.mn.AddCredentials(context.Background(), params)
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

	result, err := handler.mn.DownloadBackup(context.Background(), backupID)
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

	result, err := handler.mn.ListBackups(context.Background(), applicationID)
	if err != nil {
		badRequest(w, err)
		return
	}

	ok(w, "success", result)
}

func (handler *ApiHandler) ListApplications(w http.ResponseWriter, r *http.Request) {
	apps, err := handler.mn.ListApplications(context.Background())
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

	deps, err := handler.mn.ListDeployments(context.Background(), applicationID)
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	values, err := handler.mn.ListVariables(ctx, applicationID, &environment)
	if err != nil {
		serverError(w, err)
		return
	}

	ok(w, "success", values)
}
