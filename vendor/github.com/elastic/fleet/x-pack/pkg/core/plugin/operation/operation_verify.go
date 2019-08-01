// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import "github.com/pkg/errors"

// operationVerify verifies downloaded artifact for correct signature
// skips if artifact is already installed
type operationVerify struct {
}

func newOperationVerify() *operationVerify {
	return &operationVerify{}
}

// Name is human readable name identifying an operation
func (o *operationVerify) Name() string {
	return "operation-verify"
}

// Check checks whether operation needs to be run
// examples:
// - Start does not need to run if process is running
// - Fetch does not need to run if package is already present
func (o *operationVerify) Check() (bool, error) {
	return false, nil
}

// Run runs the operation
func (o *operationVerify) Run() (err error) {
	defer func() {
		if err != nil {
			err = errors.Wrap(err, o.Name())
		}
	}()

	return nil
}
