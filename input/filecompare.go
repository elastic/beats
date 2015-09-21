// +build !windows

package input

import (
	"github.com/elastic/libbeat/logp"
	"os"
)

// SafeFileRotate safely rotates an existing file under path and replaces it with the tempfile
func SafeFileRotate(path, tempfile string) error {
	if e := os.Rename(tempfile, path); e != nil {
		logp.Info("Registry rotate error: rename of %s to %s - Error: %s", tempfile, path, e)
		return e
	}
	return nil
}
