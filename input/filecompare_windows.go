package input

import (
	"github.com/elastic/libbeat/logp"
	"os"
)

func IsSameFile(path string, info os.FileInfo, state *FileState) bool {
	// Do we have any other way to validate a file is the same file
	// under windows?
	return path == *state.Source
}

func (f1 *File) IsSameFile(f2 *File) bool {
	// TODO: Anything meaningful to compare on file infos?
	return true
}

func fileIds(info *os.FileInfo) (uint64, uint64) {
	// No dev and inode numbers on windows, right?
	return 0, 0
}

// SafeFileRotate safely rotates an existing file under path and replaces it with the tempfile
func SafeFileRotate(path, tempfile string) error {
	old := path + ".old"
	var e error

	if e = os.Rename(path, old); e != nil {
		logp.Info("rotate: rename of %s to %s - %s", path, old, e)
		return e
	}

	if e = os.Rename(tempfile, path); e != nil {
		logp.Info("rotate: rename of %s to %s - %s", tempfile, path, e)
		return e
	}
	return nil
}
