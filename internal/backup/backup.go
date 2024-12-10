package backup

import (
	"context"
	"sarabi/internal/storage"
	types "sarabi/internal/types"
)

type (
	ExecuteParams struct {
		Environment       string
		DatabaseVars      []*types.Secret
		StorageCredential *types.StorageCredentials
		Application       *types.Application
	}

	ExecuteResponse struct {
		Location    string
		StorageType storage.Type
	}

	Executor interface {
		Execute(ctx context.Context, params ExecuteParams) (ExecuteResponse, error)
	}
)
