// +build darwin openbsd

package input

import (
	"os"
	"syscall"
)

// Returns Inode and Device
func fileIds(info *os.FileInfo) (uint64, int32) {
	fstat := (*(info)).Sys().(*syscall.Stat_t)
	return fstat.Ino, fstat.Dev
}

// This is the struct used to be sent between channels for communication
type FileState struct {
	Source *string `json:"source,omitempty"`
	Offset int64   `json:"offset,omitempty"`
	Inode  uint64  `json:"inode,omitempty"`
	Device int32   `json:"device,omitempty"`
}
