// +build !darwin,!freebsd,!linux,!openbsd,!windows

package gosigar

import (
	"runtime"
)

func (c *Cpu) Get() error {
	return ErrNotImplemented{runtime.GOOS}
}

func (l *LoadAverage) Get() error {
	return ErrNotImplemented{runtime.GOOS}
}

func (m *Mem) Get() error {
	return ErrNotImplemented{runtime.GOOS}
}

func (s *Swap) Get() error {
	return ErrNotImplemented{runtime.GOOS}
}

func (f *FDUsage) Get() error {
	return ErrNotImplemented{runtime.GOOS}
}

func (p *ProcTime) Get(int) error {
	return ErrNotImplemented{runtime.GOOS}
}

func (self *FileSystemUsage) Get(path string) error {
	return ErrNotImplemented{runtime.GOOS}
}
