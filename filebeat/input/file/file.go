package file

import (
	"os"

	"github.com/elastic/beats/libbeat/logp"
)

type File struct {
	File     *os.File
	FileInfo os.FileInfo
	Path     string
	State    *State
}

// Checks if the two files are the same.
func (f *File) IsSameFile(f2 *File) bool {
	return os.SameFile(f.FileInfo, f2.FileInfo)
}

// IsSameFile checks if the given File path corresponds with the FileInfo given
func IsSameFile(path string, info os.FileInfo) bool {
	fileInfo, err := os.Stat(path)

	if err != nil {
		logp.Err("Error during file comparison: %s with %s - Error: %s", path, info.Name(), err)
		return false
	}

	return os.SameFile(fileInfo, info)
}
