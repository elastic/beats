// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

package install

// fixPermissions fixes the permissions to be correct on the installed system
func fixPermissions() error {
	// do nothing at the moment
	return nil
}
