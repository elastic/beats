package input

import (
	"github.com/elastic/libbeat/logp"
	"os"
)

func fileIds(info *os.FileInfo) (uint64, uint64) {
	// TODO File id and device seem to exist: https://github.com/golang/go/blob/master/src/os/stat_windows.go#L43
	//https://github.com/golang/go/blob/master/src/os/types_windows.go#L14
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
