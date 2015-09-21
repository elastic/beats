// +build !windows

package input

import (
	"os"
	"syscall"

	"github.com/elastic/libbeat/logp"
)

// IsSameFile checks if the given File path corresponds with the FileInfo given
func IsSameFile(path string, info os.FileInfo) bool {
	fileInfo, err := os.Stat(path)

	if err != nil {
		logp.Info("Error during file comparison: %s with %s", path, info.Name())
		return false
	}

	return os.SameFile(fileInfo, info)
}

// Checks if the two files are the same.
func (f1 *File) IsSameFile(f2 *File) bool {
	return os.SameFile(f1.FileInfo, f2.FileInfo)
}

// Compare file stats. Inode id and device are compared
func compareFileStats(s1 *syscall.Stat_t, s2 *syscall.Stat_t) bool {
	return (s1.Dev == s2.Dev && s1.Ino == s2.Ino)
}

// SafeFileRotate safely rotates an existing file under path and replaces it with the tempfile
func SafeFileRotate(path, tempfile string) error {
	if e := os.Rename(tempfile, path); e != nil {
		logp.Info("registry rotate: rename of %s to %s - %s", tempfile, path, e)
		return e
	}
	return nil
}
