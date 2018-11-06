// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows !cgo

package user

import (
	"github.com/pkg/errors"
)

// GetUsers is not implemented on all systems.
func GetUsers() (users []*User, err error) {
	return nil, errors.New("not implemented")
}
