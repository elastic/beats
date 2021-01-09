// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows

package install

import (
	"os"
)

const (
	// PermissionUser is the permission level the user needs to be.
	PermissionUser = "Administrator"
)

// HasRoot returns true if the user has Administrator/SYSTEM permissions.
func HasRoot() bool {
	// only valid rights can open the physical drive
	f, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}
	defer f.Close()
	return true
}
