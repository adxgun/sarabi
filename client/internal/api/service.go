package api

import "context"

type (
	Service interface {
		ApplicationService
	}

	ApplicationService interface {
		CreateApplication(ctx context.Context, params CreateApplicationParams) (Application, error)
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
