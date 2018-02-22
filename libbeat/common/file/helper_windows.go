package file

import (
	"os"
)

// SafeFileRotate safely rotates an existing file under path and replaces it with the tempfile
func SafeFileRotate(path, tempfile string) error {
	old := path + ".old"
	var e error

	// In Windows, one cannot rename a file if the destination already exists, at least
	// not with using the os.Rename function that Golang offers.
	// This tries to move the existing file into an old file first and only do the
	// move after that.
	if e = os.Remove(old); e != nil {
		// ignore error in case old doesn't exit yet
	}
	if e = os.Rename(path, old); e != nil {
		// ignore error in case path doesn't exist
	}

	if e = os.Rename(tempfile, path); e != nil {
		return e
	}
	return nil
}
