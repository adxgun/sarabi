package api

import (
	"context"
	"io"
)

type (
	Service interface {
		ApplicationService
	}

	ApplicationService interface {
		CreateApplication(ctx context.Context, params CreateApplicationParams) (Application, error)
		Deploy(ctx context.Context, frontend, backend io.Reader, params DeployParams) (DeployResponse, error)
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
