// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package rollback

import "context"

// ErrorChecker checks agent for status change and sends an error to a channel if found.
type ErrorChecker struct {
	notifyChan chan error
}

// NewErrorChecker creates a new error checker.
func NewErrorChecker(ch chan error) *ErrorChecker {
	return &ErrorChecker{
		notifyChan: ch,
	}
}

// Run runs the checking loop.
func (ch ErrorChecker) Run(ctx context.Context) {
	// TODO: finish me
}
