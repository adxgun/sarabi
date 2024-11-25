package storage

import (
	"context"
	"io"
)

type (
	Storage interface {
		Save(ctx context.Context, location string, content io.Reader) error
		Get(ctx context.Context, location string) (io.Reader, error)
	}
)
