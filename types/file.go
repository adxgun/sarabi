package types

import "io"

type File struct {
	Content io.Reader
	Stat    FileStat
}

type FileStat struct {
	Size int64
	Name string
}
