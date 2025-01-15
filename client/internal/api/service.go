package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
)

type (
	Service interface {
		ApplicationService
		BackupService
	}

	ApplicationService interface {
		CreateApplication(ctx context.Context, params CreateApplicationParams) (Application, error)
		Deploy(ctx context.Context, frontend, backend io.Reader, params DeployParams) (<-chan Event, error)
		UpdateVariables(ctx context.Context, applicationID uuid.UUID, params UpdateVariablesParams) error
		ListApplications(ctx context.Context) ([]Application, error)
		Destroy(ctx context.Context, applicationID uuid.UUID, environment string) error
		ListVariables(ctx context.Context, applicationID uuid.UUID, environment string) ([]Var, error)
		AddDomain(ctx context.Context, applicationID uuid.UUID, param AddDomainParam) error
		RemoveDomain(ctx context.Context, applicationID uuid.UUID, name string) error
		ListDeployments(ctx context.Context, applicationID uuid.UUID) ([]Deployment, error)
		Scale(ctx context.Context, applicationID uuid.UUID, params ScaleAppParams) error
		Rollback(ctx context.Context, identifier string) error
		TailLogs(ctx context.Context, applicationID uuid.UUID, environment string) (io.ReadCloser, error)
	}

	BackupService interface {
		CreateBackupSchedule(ctx context.Context, applicationID uuid.UUID, params CreateBackupParams) error
		ListBackups(ctx context.Context, applicationID uuid.UUID, environment string) ([]Backup, error)
		DownloadBackup(ctx context.Context, backupID uuid.UUID) (io.ReadCloser, error)
	}
)

type service struct {
	apiClient Client
}

func NewService(apiClient Client) Service {
	return service{apiClient: apiClient}
}

func (s service) CreateApplication(ctx context.Context, params CreateApplicationParams) (Application, error) {
	var response struct {
		Application Application `json:"data"`
	}

	err := s.apiClient.Do(ctx, Params{
		Method:   "POST",
		Path:     "applications",
		Body:     params,
		Response: &response,
	})
	return response.Application, err
}

func (s service) Deploy(ctx context.Context, frontend, backend io.Reader, params DeployParams) (<-chan Event, error) {
	files := make([]MultipartFile, 0)
	if frontend != nil {
		files = append(files, MultipartFile{
			Content: frontend,
			Name:    "frontend.tar.gz",
		})
	}
	if backend != nil {
		files = append(files, MultipartFile{
			Content: backend,
			Name:    "backend.tar.gz",
		})
	}

	var response struct {
		Data DeployResponse `json:"data"`
	}
	httpParams := Params{
		Method:   "POST",
		Path:     "deploy",
		Body:     params,
		Response: &response,
	}

	ch := make(chan Event, 100)
	go func() {
		resp, err := s.apiClient.DoMultipart(ctx, files, httpParams)
		if err != nil {
			ch <- Event{
				Type:    Error,
				Message: err.Error(),
			}
			return
		}

		sc := bufio.NewScanner(resp)
		for sc.Scan() {
			ev := &Event{}
			if err := json.Unmarshal(sc.Bytes(), ev); err != nil {
				continue
			}

			ch <- *ev
		}

		if err := sc.Err(); err != nil {
		}
	}()
	return ch, nil
}

func (s service) UpdateVariables(ctx context.Context, applicationID uuid.UUID, params UpdateVariablesParams) error {
	var response struct {
		Message string `json:"message"`
	}

	url := fmt.Sprintf("applications/%s/variables", applicationID)
	param := Params{
		Method:   "PUT",
		Path:     url,
		Body:     params,
		Response: &response,
	}
	return s.apiClient.Do(ctx, param)
}

func (s service) ListApplications(ctx context.Context) ([]Application, error) {
	var response struct {
		Data []Application `json:"data"`
	}

	err := s.apiClient.Do(ctx, Params{
		Method:   "GET",
		Path:     "applications",
		Response: &response,
	})
	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (s service) Destroy(ctx context.Context, applicationID uuid.UUID, environment string) error {
	type body struct {
		Environment string `json:"environment"`
	}

	var response struct {
		Message string `json:"message"`
	}

	b := body{Environment: environment}
	param := Params{
		Method:   "POST",
		Path:     fmt.Sprintf("applications/%s/destroy", applicationID),
		Body:     b,
		Response: &response,
	}
	return s.apiClient.Do(ctx, param)
}

func (s service) ListVariables(ctx context.Context, applicationID uuid.UUID, environment string) ([]Var, error) {
	var response struct {
		Data []Var `json:"data"`
	}

	u := fmt.Sprintf("applications/%s/variables", applicationID)
	param := Params{
		Method:   "GET",
		Path:     u,
		Response: &response,
		QueryParams: map[string]string{
			"environment": environment,
		},
	}

	err := s.apiClient.Do(ctx, param)
	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (s service) AddDomain(ctx context.Context, applicationID uuid.UUID, param AddDomainParam) error {
	var response struct {
		Message string `json:"message"`
	}
	params := Params{
		Method:   "PUT",
		Path:     fmt.Sprintf("applications/%s/domains", applicationID),
		Body:     param,
		Response: &response,
	}
	return s.apiClient.Do(ctx, params)
}

func (s service) RemoveDomain(ctx context.Context, applicationID uuid.UUID, name string) error {
	var response struct {
		Message string `json:"message"`
	}

	type body struct {
		Name string `json:"name"`
	}
	b := body{Name: name}

	params := Params{
		Method:   "DELETE",
		Path:     fmt.Sprintf("applications/%s/domains", applicationID),
		Body:     b,
		Response: &response,
	}
	return s.apiClient.Do(ctx, params)
}

func (s service) ListDeployments(ctx context.Context, applicationID uuid.UUID) ([]Deployment, error) {
	var response struct {
		Data []Deployment `json:"data"`
	}

	u := fmt.Sprintf("applications/%s/deployments", applicationID)
	param := Params{
		Method:   "GET",
		Path:     u,
		Response: &response,
	}

	err := s.apiClient.Do(ctx, param)
	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (s service) Scale(ctx context.Context, applicationID uuid.UUID, params ScaleAppParams) error {
	u := fmt.Sprintf("applications/%s/scale", applicationID)
	param := Params{
		Method: "PATCH",
		Path:   u,
		Body:   params,
	}

	err := s.apiClient.Do(ctx, param)
	if err != nil {
		return err
	}
	return nil
}

func (s service) Rollback(ctx context.Context, identifier string) error {
	params := RollbackParams{Identifier: identifier}
	param := Params{
		Method: "PATCH",
		Path:   "applications/rollback",
		Body:   params,
	}

	err := s.apiClient.Do(ctx, param)
	if err != nil {
		return err
	}
	return nil
}

func (s service) CreateBackupSchedule(ctx context.Context, applicationID uuid.UUID, params CreateBackupParams) error {
	param := Params{
		Method: "PUT",
		Path:   fmt.Sprintf("applications/%s/backup-settings", applicationID),
		Body:   params,
	}

	err := s.apiClient.Do(ctx, param)
	if err != nil {
		return err
	}
	return nil
}

func (s service) ListBackups(ctx context.Context, applicationID uuid.UUID, environment string) ([]Backup, error) {
	var response struct {
		Data []Backup `json:"data"`
	}
	param := Params{
		Method:   "GET",
		Path:     fmt.Sprintf("applications/%s/backups", applicationID),
		Response: &response,
		QueryParams: map[string]string{
			"environment": environment,
		},
	}
	if err := s.apiClient.Do(ctx, param); err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (s service) DownloadBackup(ctx context.Context, backupID uuid.UUID) (io.ReadCloser, error) {
	param := Params{
		Method: "GET",
		Path:   fmt.Sprintf("backups/%s/download", backupID),
	}
	return s.apiClient.Download(ctx, param)
}

func (s service) TailLogs(ctx context.Context, applicationID uuid.UUID, environment string) (io.ReadCloser, error) {
	param := Params{
		Method:      "GET",
		Path:        fmt.Sprintf("applications/%s/logs", applicationID),
		QueryParams: map[string]string{"environment": environment},
	}
	return s.apiClient.Download(ctx, param)
}
