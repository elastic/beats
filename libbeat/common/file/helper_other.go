// +build !windows

package file

import (
	"os"
	"path/filepath"
)

// SafeFileRotate safely rotates an existing file under path and replaces it with the tempfile
func SafeFileRotate(path, tempfile string) error {
	parent := filepath.Dir(path)

	if e := os.Rename(tempfile, path); e != nil {
		return e
	}

	// best-effort fsync on parent directory. The fsync is required by some
	// filesystems, so to update the parents directory metadata to actually
	// contain the new file being rotated in.
	f, err := os.Open(parent)
	if err != nil {
		return nil // ignore error
	}
	defer f.Close()
	f.Sync()

	return nil
}
