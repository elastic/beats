// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package txfile

import (
	"golang.org/x/sys/unix"
)

type osFileState struct{}

func (f *osFile) MMap(sz int) ([]byte, error) {
	return unix.Mmap(int(f.Fd()), 0, int(sz), unix.PROT_READ, unix.MAP_SHARED)
}

func (f *osFile) MUnmap(b []byte) error {
	return unix.Munmap(b)
}

func (f *osFile) Lock(exclusive, blocking bool) error {
	flags := unix.LOCK_SH
	if exclusive {
		flags = unix.LOCK_EX
	}
	if !blocking {
		flags |= unix.LOCK_NB
	}

	return unix.Flock(int(f.Fd()), flags)
}

func (f *osFile) Unlock() error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}
