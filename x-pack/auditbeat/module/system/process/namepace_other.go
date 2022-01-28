// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !linux
// +build !linux

package process

// isNsSharedWith returns true and nil.
func isNsSharedWith(pid int, ns string) (yes bool, err error) {
	return true, nil
}
