// +build !windows

package file

import (
	"os"

	"github.com/elastic/beats/libbeat/logp"
)

// SafeFileRotate safely rotates an existing file under path and replaces it with the tempfile
func SafeFileRotate(path, tempfile string) error {
	if e := os.Rename(tempfile, path); e != nil {
		logp.Err("Rotate error: %s", e)
		return e
	}
	return nil
}
