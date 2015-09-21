package input

import (
	"github.com/elastic/libbeat/logp"
	"os"
)

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
