// +build !windows

package file

import (
	"os"
	"strconv"
	"syscall"
)

type StateOS struct {
	Inode  uint64 `json:"inode,"`
	Device uint64 `json:"device,"`
}

// GetOSState returns the FileStateOS for non windows systems
func GetOSState(info os.FileInfo) StateOS {
	stat := info.Sys().(*syscall.Stat_t)

	// Convert inode and dev to uint64 to be cross platform compatible
	fileState := StateOS{
		Inode:  uint64(stat.Ino),
		Device: uint64(stat.Dev),
	}

	return fileState
}

// IsSame file checks if the files are identical
func (fs StateOS) IsSame(state StateOS) bool {
	return fs.Inode == state.Inode && fs.Device == state.Device
}

func (fs StateOS) String() string {
	var buf [64]byte
	current := strconv.AppendUint(buf[:0], fs.Inode, 10)
	current = append(current, '-')
	current = strconv.AppendUint(current, fs.Device, 10)
	return string(current)
}

// ReadOpen opens a file for reading only
func ReadOpen(path string) (*os.File, error) {
	flag := os.O_RDONLY
	perm := os.FileMode(0)
	return os.OpenFile(path, flag, perm)
}
