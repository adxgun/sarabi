package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
)

type fileStorage struct {
}

func NewFileStorage() Storage {
	return &fileStorage{}
}

func (f fileStorage) Save(ctx context.Context, location string, content io.Reader) error {
	dir := filepath.Dir(location)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	fi, err := os.Create(location)
	if err != nil {
		return err
	}

	_, err = io.Copy(fi, content)
	if err != nil {
		return err
	}
	return nil
}

func (f fileStorage) Get(ctx context.Context, location string) (io.Reader, error) {
	fi, err := os.Open(location)
	if err != nil {
		return nil, err
	}

	content, err := io.ReadAll(fi)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(content), nil
}
