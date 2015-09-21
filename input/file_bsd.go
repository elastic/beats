// +build darwin openbsd

package input

import (
	"os"
	"syscall"
)

// Returns Inode and Device
func fileIds(info *os.FileInfo) (uint64, uint64) {
	fstat := (*(info)).Sys().(*syscall.Stat_t)
	return fstat.Ino, uint64(fstat.Dev)
}
