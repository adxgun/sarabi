package api

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"io"
)

type (
	Service interface {
		ApplicationService
	}

	ApplicationService interface {
		CreateApplication(ctx context.Context, params CreateApplicationParams) (Application, error)
		Deploy(ctx context.Context, frontend, backend io.Reader, params DeployParams) (DeployResponse, error)
		UpdateVariables(ctx context.Context, applicationID uuid.UUID, params UpdateVariablesParams) error
		ListApplications(ctx context.Context) ([]Application, error)
		Destroy(ctx context.Context, applicationID uuid.UUID, environment string) error
		ListVariables(ctx context.Context, applicationID uuid.UUID, environment string) ([]Var, error)
		AddDomain(ctx context.Context, applicationID uuid.UUID, param AddDomainParam) error
		RemoveDomain(ctx context.Context, applicationID uuid.UUID, name string) error
		ListDeployments(ctx context.Context, applicationID uuid.UUID) ([]Deployment, error)
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

func (s service) Deploy(ctx context.Context, frontend, backend io.Reader, params DeployParams) (DeployResponse, error) {
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

	err := s.apiClient.DoMultipart(ctx, files, httpParams)
	if err != nil {
		return DeployResponse{}, err
	}

	return response.Data, nil
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
