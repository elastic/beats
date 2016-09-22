// +build !windows

package checkpoint

import "os"

func create(path string) (*os.File, error) {
	return os.Create(path)
}
