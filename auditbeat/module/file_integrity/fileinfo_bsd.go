// +build freebsd openbsd netbsd darwin

package file_integrity

import (
	"syscall"
	"time"
)

func fileTimes(stat *syscall.Stat_t) (atime, mtime, ctime time.Time) {
	return time.Unix(0, stat.Atimespec.Nano()).UTC(),
		time.Unix(0, stat.Mtimespec.Nano()).UTC(),
		time.Unix(0, stat.Mtimespec.Nano()).UTC()
}
