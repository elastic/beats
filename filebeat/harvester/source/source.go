package source

import (
    "io"
    "os"
)

type LogSource interface {
    io.ReadCloser
    Name() string
    io.Seeker
}

type FileSource interface {
    LogSource
    Stat() (os.FileInfo, error)
    Continuable() bool // can we continue processing after EOF?
    ActualSize() (int64, error)
}
