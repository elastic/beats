// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows
// +build !windows

package install

// postInstall performs post installation for unix-based systems.
func postInstall() error {
	// do nothing
	return nil
}
