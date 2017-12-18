// +build linux

package file_integrity

import (
	"syscall"
	"time"
)

func fileTimes(stat *syscall.Stat_t) (atime, mtime, ctime time.Time) {
	return time.Unix(0, stat.Atim.Nano()).UTC(),
		time.Unix(0, stat.Mtim.Nano()).UTC(),
		time.Unix(0, stat.Ctim.Nano()).UTC()
}
