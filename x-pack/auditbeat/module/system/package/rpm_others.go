// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (!linux || !cgo) && !windows

package pkg

import "errors"

func listRPMPackages(_ bool) ([]*Package, error) {
	return nil, errors.New("listing RPM packages is only supported on Linux")
}

func closeDataset() error {
	return nil
}
