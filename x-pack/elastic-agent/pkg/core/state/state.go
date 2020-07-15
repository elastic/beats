// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package state

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/process"
)

// Status describes the current status of the application process.
type Status int

const (
	// Stopped is status describing not running application.
	Stopped Status = iota
	// Starting is status describing application is starting.
	Starting
	// Configuring is status describing application is configuring.
	Configuring
	// Running is status describing application is running.
	Running
	// Degraded is status describing application is degraded.
	Degraded
	// Failed is status describing application is failed.
	Failed
	// Stopping is status describing application is stopping.
	Stopping
	// Crashed is status describing application is crashed.
	Crashed
	// Restarting is status describing application is restarting.
	Restarting
)

// State wraps the process state and application status.
type State struct {
	ProcessInfo *process.Info
	Status      Status
	Message     string
	Payload     map[string]interface{}
}

// Reporter is interface that is called when a state is changed.
type Reporter interface {
	// OnStateChange is called when state changes.
	OnStateChange(id string, name string, state State)
}
