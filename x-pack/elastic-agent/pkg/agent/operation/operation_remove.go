// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/state"
)

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

// Check checks whether remove needs to run.
//
// Always returns false.
func (o *operationRemove) Check(_ context.Context, _ Application) (bool, error) {
	return false, nil
}

// Run runs the operation
func (o *operationRemove) Run(ctx context.Context, application Application) (err error) {
	defer func() {
		if err != nil {
			application.SetState(state.Failed, err.Error(), nil)
		}
	}()

	return nil
}
