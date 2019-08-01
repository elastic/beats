// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import "github.com/pkg/errors"

// operationRemove uninstall and removes all the bits related to the artifact
type operationRemove struct {
}

func newOperationRemove() *operationRemove {
	return &operationRemove{}
}

// Name is human readable name identifying an operation
func (o *operationRemove) Name() string {
	return "operation-remove"
}

// Check checks whether operation needs to be run
// examples:
// - Start does not need to run if process is running
// - Fetch does not need to run if package is already present
func (o *operationRemove) Check() (bool, error) {
	return false, nil
}

// Run runs the operation
func (o *operationRemove) Run() (err error) {
	defer func() {
		if err != nil {
			err = errors.Wrap(err, o.Name())
		}
	}()

	return nil
}
