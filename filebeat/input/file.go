package input

import (
	"os"

	"github.com/elastic/beats/libbeat/logp"
)

const (
	jsonErrorKey = "json_error"
)

type File struct {
	File      *os.File
	FileInfo  os.FileInfo
	Path      string
	FileState *FileState
}

// Check that the file isn't a symlink, mode is regular or file is nil
func (f *File) IsRegularFile() bool {
	if f.File == nil {
		logp.Critical("Harvester: BUG: f arg is nil")
		return false
	}

	info, e := f.File.Stat()
	if e != nil {
		logp.Err("File check fault: stat error: %s", e.Error())
		return false
	}

	if !info.Mode().IsRegular() {
		logp.Warn("Harvester: not a regular file: %q %s", info.Mode(), info.Name())
		return false
	}
	return true
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

func IsRegularFile(file *os.File) bool {
	f := &File{File: file}
	return f.IsRegularFile()
}
