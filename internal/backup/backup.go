package backup

import (
	"context"
	"sarabi/internal/storage"
	types "sarabi/internal/types"
)

type (
	Params struct {
		Environment       string
		DatabaseVars      []*types.Secret
		StorageCredential *types.StorageCredentials
		Application       *types.Application
	}

	Result struct {
		Location    string
		StorageType storage.Type
	}

	Executor interface {
		Execute(ctx context.Context, params Params) (Result, error)
	}
)
