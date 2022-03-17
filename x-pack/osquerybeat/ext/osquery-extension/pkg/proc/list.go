// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package proc

import (
	"io/fs"
	"os"
	"strconv"
)

// List returns all the processes in the proc folder
func List(root string) ([]string, error) {
	return ListFS(os.DirFS(root))
}

func ListFS(sysfs fs.FS) ([]string, error) {
	var pids []string

	dirs, err := fs.ReadDir(sysfs, "proc")

	if err != nil {
		return nil, err
	}

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		name := dir.Name()
		// Check if directory is number
		_, err := strconv.Atoi(name)
		if err != nil {
			err = nil
			continue
		}
		pids = append(pids, name)
	}

	return pids, nil
}
