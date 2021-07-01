// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

package install

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
)

// fixPermissions fixes the permissions so only root:root is the owner and no world read-able permissions
func fixPermissions() error {
	return recursiveRootPermissions(paths.InstallPath)
}

func recursiveRootPermissions(path string) error {
	return filepath.Walk(path, func(name string, info fs.FileInfo, err error) error {
		if err == nil {
			// all files should be owned by root:root
			err = os.Chown(name, 0, 0)
			if err != nil {
				return err
			}
			// remove any world permissions from the file
			err = os.Chmod(name, info.Mode().Perm()&0770)
		}
		return err
	})
}
