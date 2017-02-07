package file

import (
	"os"
)

func stat(name string, statFunc func(name string) (os.FileInfo, error)) (FileInfo, error) {
	info, err := statFunc(name)
	if err != nil {
		return nil, err
	}

	return fileInfo{FileInfo: info}, nil
}
