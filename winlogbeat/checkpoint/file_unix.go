// +build !windows

package checkpoint

import "os"

func create(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_SYNC, 0600)
}
