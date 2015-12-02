// +build !windows

package input

import (
	"os"
	"syscall"

	"github.com/elastic/libbeat/logp"
)

type FileStateOS struct {
	Inode  uint64 `json:"inode,omitempty"`
	Device uint64 `json:"device,omitempty"`
}

// GetOSFileState returns the FileStateOS for non windows systems
func GetOSFileState(info *os.FileInfo) *FileStateOS {

	stat := (*(info)).Sys().(*syscall.Stat_t)

	// Convert inode and dev to uint64 to be cross platform compatible
	fileState := &FileStateOS{
		Inode:  uint64(stat.Ino),
		Device: uint64(stat.Dev),
	}

	return fileState
}

// IsSame file checks if the files are identical
func (fs *FileStateOS) IsSame(state *FileStateOS) bool {
	return fs.Inode == state.Inode && fs.Device == state.Device
}

// SafeFileRotate safely rotates an existing file under path and replaces it with the tempfile
func SafeFileRotate(path, tempfile string) error {
	if e := os.Rename(tempfile, path); e != nil {
		logp.Err("Rotate error: %s", e)
		return e
	}
	return nil
}

// ReadOpen opens a file for reading only
func ReadOpen(path string) (*os.File, error) {

	flag := os.O_RDONLY
	var perm os.FileMode = 0

	return os.OpenFile(path, flag, perm)
}
