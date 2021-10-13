// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !linux && !darwin
// +build !linux,!darwin

package app

// UserGroup returns the uid and gid for the process specification.
func (spec ProcessSpec) UserGroup() (int, int, error) {
	return 0, 0, nil
}
