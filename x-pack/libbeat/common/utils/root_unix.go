// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows

package utils

import "os"

const (
	// PermissionUser is the permission level the user needs to be.
	PermissionUser = "root"
)

// HasRoot returns true if the user has root permissions.
// Added extra `nil` value to return since the HasRoot for windows will return an error as well
func HasRoot() (bool, error) {
	return os.Geteuid() == 0, nil
}
