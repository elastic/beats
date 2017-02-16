package file

import (
	"errors"
	"os"
)

// A FileInfo describes a file and is returned by Stat and Lstat.
type FileInfo interface {
	os.FileInfo
	UID() (int, error) // UID of the file owner. Returns an error on non-POSIX file systems.
	GID() (int, error) // GID of the file owner. Returns an error on non-POSIX file systems.
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *PathError.
func Stat(name string) (FileInfo, error) {
	return stat(name, os.Stat)
}

// Lstat returns a FileInfo describing the named file.
// If the file is a symbolic link, the returned FileInfo
// describes the symbolic link. Lstat makes no attempt to follow the link.
// If there is an error, it will be of type *PathError.
func Lstat(name string) (FileInfo, error) {
	return stat(name, os.Lstat)
}

type fileInfo struct {
	os.FileInfo
	uid *int
	gid *int
}

func (f fileInfo) UID() (int, error) {
	if f.uid == nil {
		return -1, errors.New("uid not implemented")
	}

	return *f.uid, nil
}

func (f fileInfo) GID() (int, error) {
	if f.gid == nil {
		return -1, errors.New("gid not implemented")
	}

	return *f.gid, nil
}
