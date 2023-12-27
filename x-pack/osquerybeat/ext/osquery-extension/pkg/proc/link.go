// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package proc

import (
	"os"
	"path/filepath"
)

func ReadLink(root string, pid string, attr string) (string, error) {
	fn := filepath.Join(root, pid, attr)

	s, err := os.Readlink(fn)
	if err != nil {
		return "", err
	}
	return s, nil
}
