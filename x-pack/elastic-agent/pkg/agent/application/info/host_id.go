// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !linux
// +build !windows
// +build !darwin
// +build !freebsd
// +build !netbsd
// +build !openbsd

package info

import "errors"

// loadHostID will return an error on systems that do not have a specific implementation.
func loadHostID() (string, error) {
	return "", errors.New("loadHostID unimplemented")
}
