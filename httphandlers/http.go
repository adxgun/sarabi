package httphandlers

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net/http"
	"sarabi/logger"
	"sarabi/manager"
	"sarabi/types"
	"strings"
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

	deployParamsPayload := r.FormValue("json")
	logger.Info("body", zap.String("json", deployParamsPayload))
	if err := json.Unmarshal([]byte(deployParamsPayload), &body); err != nil {
		badRequest(w, errors.Wrap(err, "invalid request body"))
		return
	}

	param := &types.DeployParams{
		ApplicationID: body.ApplicationID,
		Instances:     body.Instances,
		Environment:   body.Environment,
	}
	for _, ff := range r.MultipartForm.File["files"] {
		logger.Info("file: ",
			zap.String("name", ff.Filename),
			zap.Int64("size", ff.Size))
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

	logger.Info("starting deployment", zap.Any("params", param.ApplicationID))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resp, err := handler.mn.Deploy(ctx, param)
	if err != nil {
		serverError(w, errors.Wrap(err, "deployment failed"))
		return
	}

	ok(w, "deployment succeeded", resp)
}
