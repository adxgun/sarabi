package types

type File struct {
	Content []byte
	Stat    FileStat
}

type FileStat struct {
	Size int64
	Name string
}
