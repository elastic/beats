// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package state

import "github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/process"

// Status describes the current status of the application process.
type Status int

const (
	// Stopped is status describing not running application.
	Stopped Status = iota
	// Running signals that application is currently running.
	Running
	// Restarting means process crashed and is being started again.
	Restarting
)

// State wraps the process state and application status.
type State struct {
	ProcessInfo *process.Info
	Status      Status
}
