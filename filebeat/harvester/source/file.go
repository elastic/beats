package source

import "os"

type File struct {
	*os.File
}

func (File) Continuable() bool { return true }
