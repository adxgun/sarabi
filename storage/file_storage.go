package storage

import (
	"bufio"
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

	writer := bufio.NewWriter(fi)
	buffer := make([]byte, bufferSize)
	for {
		n, err := file.Content.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if n > 0 {
			_, err = writer.Write(buffer[:n])
			if err != nil {
				return err
			}
		}
	}

	if err := writer.Flush(); err != nil {
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

	return &types.File{
		Content: fi,
		Stat:    types.FileStat{Size: stat.Size(), Name: stat.Name()},
	}, nil
}
