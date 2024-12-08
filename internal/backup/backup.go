package backup

import (
	"context"
	"sarabi/internal/storage"
	types2 "sarabi/internal/types"
)

type (
	ExecuteParams struct {
		Environment       string
		DatabaseVars      []*types2.Secret
		StorageCredential *types2.StorageCredentials
		Application       *types2.Application
	}

	ExecuteResponse struct {
		Location    string
		StorageType storage.Type
	}

	Executor interface {
		Execute(ctx context.Context, params ExecuteParams) (ExecuteResponse, error)
	}
)
