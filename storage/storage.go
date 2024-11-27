package storage

import (
	"context"
	"io"
	"sarabi/types"
)

type (
	Storage interface {
		Save(ctx context.Context, location string, f types.File) error
		Get(ctx context.Context, location string) (io.Reader, error)
	}
)
