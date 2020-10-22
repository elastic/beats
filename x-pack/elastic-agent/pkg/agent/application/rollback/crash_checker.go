// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package rollback

import "context"

// CrashChecker checks agent for crash pattern in Elastic Agent lifecycle.
type CrashChecker struct {
	notifyChan chan error
}

// NewCrashChecker creates a new crash checker.
func NewCrashChecker(ch chan error) *CrashChecker {
	return &CrashChecker{
		notifyChan: ch,
	}
}

// Run runs the checking loop.
func (ch CrashChecker) Run(ctx context.Context) {
	// TODO: finish me

}

func getAgentServicePid() int {
	// TODO: finish me
	return 0
}
