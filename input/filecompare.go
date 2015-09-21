// +build !windows

package input

import (
	"os"
	"syscall"

	"github.com/elastic/libbeat/logp"
)

// SafeFileRotate safely rotates an existing file under path and replaces it with the tempfile
func SafeFileRotate(path, tempfile string) error {
	if e := os.Rename(tempfile, path); e != nil {
		logp.Info("registry rotate: rename of %s to %s - %s", tempfile, path, e)
		return e
	}
	return nil
}
