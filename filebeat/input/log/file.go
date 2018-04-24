package log

import "os"

type File struct {
	*os.File
}

func (File) Continuable() bool { return true }
func (File) HasState() bool    { return true }
