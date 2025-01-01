package types

import (
	"io"
	"os"
)

type File struct {
	Content io.ReadCloser
	Stat    FileStat
}

type FileStat struct {
	Size        int64
	Name        string
	Mode        os.FileMode
	ContentType string
}

type NoOpReadCloser struct {
	io.Reader
}

func (NoOpReadCloser) Close() error {
	return nil
}

func (f File) GetContentType() string {
	if f.Stat.ContentType == "" {
		return "application/octet-stream"
	}
	return f.Stat.ContentType
}
