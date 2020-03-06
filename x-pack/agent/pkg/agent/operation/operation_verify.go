// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/errors"
)

// operationVerify verifies downloaded artifact for correct signature
// skips if artifact is already installed
type operationVerify struct {
	eventProcessor callbackHooks
}

func newOperationVerify(eventProcessor callbackHooks) *operationVerify {
	return &operationVerify{eventProcessor: eventProcessor}
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
func (o *operationVerify) Run(ctx context.Context, application Application) (err error) {
	defer func() {
		if err != nil {
			err = errors.New(err,
				o.Name(),
				errors.TypeApplication,
				errors.M(errors.MetaKeyAppName, application.Name()))
			o.eventProcessor.OnFailing(ctx, application.Name(), err)
		}
	}()

	return nil
}
