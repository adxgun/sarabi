package storage

import (
	"context"
	"sarabi/types"
)

type (
	Type string

	Storage interface {
		Save(ctx context.Context, location string, f types.File) error
		Get(ctx context.Context, location string) (*types.File, error)
	}
)

const (
	TypeFS Type = "File"
	TypeS3 Type = "S3"
)
