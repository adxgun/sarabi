package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"sarabi/types"
)

type fileStorage struct {
}

func NewFileStorage() Storage {
	return &fileStorage{}
}

func (f fileStorage) Save(ctx context.Context, location string, file types.File) error {
	dir := filepath.Dir(location)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	fi, err := os.Create(location)
	if err != nil {
		return err
	}

	_, err = io.Copy(fi, bytes.NewReader(file.Content))
	if err != nil {
		return err
	}
	return nil
}

func (f fileStorage) Get(ctx context.Context, location string) (*types.File, error) {
	fi, err := os.Open(location)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(location)
	if err != nil {
		return nil, err
	}

	content, err := io.ReadAll(fi)
	if err != nil {
		return nil, err
	}

	return &types.File{
		Content: content,
		Stat:    types.FileStat{Size: stat.Size(), Name: stat.Name()},
	}, nil
}
