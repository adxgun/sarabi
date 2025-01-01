package storage

import (
	"context"
	"sarabi/internal/types"
)

type (
	Type string

	Storage interface {
		Save(ctx context.Context, location string, f types.File) error
		Get(ctx context.Context, location string) (*types.File, error)
		Ping(ctx context.Context) error
	}
)

const (
	TypeFS Type = "File"
	TypeS3 Type = "S3"

	bufferSize int = 4 * 1024 * 1024 // 4MB
)

func (t Type) String() string {
	return string(t)
}
