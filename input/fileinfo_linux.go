package input

import (
	"os"
	"syscall"
)

func fileIds(info *os.FileInfo) (uint64, uint64) {
	fstat := (*info).Sys().(*syscall.Stat_t)
	return fstat.Ino, fstat.Dev
}
