// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"context"
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/retry"
)

// retryableOperations consists of multiple operations which are
// retryable as a whole.
// if nth operation fails all preceding are retried as well
type retryableOperations struct {
	logger      *logger.Logger
	operations  []operation
	retryConfig *retry.Config
}

func newRetryableOperations(
	logger *logger.Logger,
	retryConfig *retry.Config,
	operations ...operation) *retryableOperations {

	return &retryableOperations{
		logger:      logger,
		retryConfig: retryConfig,
		operations:  operations,
	}
}

// Name is human readable name identifying an operation
func (o *retryableOperations) Name() string {
	names := make([]string, 0, len(o.operations))
	for _, op := range o.operations {
		names = append(names, op.Name())
	}
	return fmt.Sprintf("retryable block: %s", strings.Join(names, " "))
}

// Check checks whether operation needs to be run
// examples:
// - Start does not need to run if process is running
// - Fetch does not need to run if package is already present
func (o *retryableOperations) Check(ctx context.Context, application Application) (bool, error) {
	for _, op := range o.operations {
		// finish early if at least one operation needs to be run or errored out
		if run, err := op.Check(ctx, application); err != nil || run {
			return run, err
		}
	}

	return false, nil
}

// Run runs the operation
func (o *retryableOperations) Run(ctx context.Context, application Application) (err error) {
	return retry.Do(ctx, o.retryConfig, o.runOnce(application))
}

// Run runs the operation
func (o *retryableOperations) runOnce(application Application) func(context.Context) error {
	return func(ctx context.Context) error {
		for _, op := range o.operations {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			shouldRun, err := op.Check(ctx, application)
			if err != nil {
				return err
			}

			if !shouldRun {
				continue
			}

			o.logger.Debugf("running operation '%s' of the block '%s'", op.Name(), o.Name())
			if err := op.Run(ctx, application); err != nil {
				o.logger.Errorf("operation %s failed, err: %v", op.Name(), err)
				return err
			}
		}

		return nil
	}
}

// check interface
var _ operation = &retryableOperations{}
