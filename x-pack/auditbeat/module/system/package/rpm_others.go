// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !linux !cgo
// +build !windows

package pkg

import "github.com/pkg/errors"

func listRPMPackages() ([]*Package, error) {
	return nil, errors.New("listing RPM packages is only supported on Linux")
}

func closeDataset() error {
	return nil
}
