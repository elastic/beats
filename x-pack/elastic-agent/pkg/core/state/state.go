// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package state

import (
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/process"
)

// Status describes the current status of the application process.
type Status int

const (
	// Stopped is status describing not running application.
	Stopped Status = -4
	// Crashed is status describing application is crashed.
	Crashed Status = -3
	// Restarting is status describing application is restarting.
	Restarting Status = -2
	// Updating is status describing application is updating.
	Updating Status = -1

	// Starting is status describing application is starting.
	Starting = Status(proto.StateObserved_STARTING)
	// Configuring is status describing application is configuring.
	Configuring = Status(proto.StateObserved_CONFIGURING)
	// Healthy is status describing application is running.
	Healthy = Status(proto.StateObserved_HEALTHY)
	// Degraded is status describing application is degraded.
	Degraded = Status(proto.StateObserved_DEGRADED)
	// Failed is status describing application is failed.
	Failed = Status(proto.StateObserved_FAILED)
	// Stopping is status describing application is stopping.
	Stopping = Status(proto.StateObserved_STOPPING)
)

// IsInternal returns true if the status is an internal status and not something that should be reported
// over the protocol as an actual status.
func (s Status) IsInternal() bool {
	return s < Starting
}

// ToProto converts the status to status that is compatible with the protocol.
func (s Status) ToProto() proto.StateObserved_Status {
	if !s.IsInternal() {
		return proto.StateObserved_Status(s)
	}
	if s == Updating || s == Restarting {
		return proto.StateObserved_STARTING
	}
	if s == Crashed {
		return proto.StateObserved_FAILED
	}
	if s == Stopped {
		return proto.StateObserved_STOPPING
	}
	// fallback to degraded
	return proto.StateObserved_DEGRADED
}

// FromProto converts the status from protocol to status Agent representation.
func FromProto(s proto.StateObserved_Status) Status {
	return Status(s)
}

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
