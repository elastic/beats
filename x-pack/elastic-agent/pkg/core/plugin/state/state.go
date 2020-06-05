// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package state

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/process"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
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
}

// UpdateFromProto updates the status from the status from the GRPC protocol.
func (s *State) UpdateFromProto(status proto.StateObserved_Status) {
	switch status {
	case proto.StateObserved_STARTING:
		s.Status = Starting
	case proto.StateObserved_CONFIGURING:
		s.Status = Configuring
	case proto.StateObserved_HEALTHY:
		s.Status = Running
	case proto.StateObserved_DEGRADED:
		s.Status = Degraded
	case proto.StateObserved_FAILED:
		s.Status = Failed
	case proto.StateObserved_STOPPING:
		s.Status = Stopping
	}
}
