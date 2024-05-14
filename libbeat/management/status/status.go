// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package status

// Status describes the current status of the beat.
type Status int

//go:generate go run golang.org/x/tools/cmd/stringer -type=Status
const (
	// Unknown is initial status when none has been reported.
	Unknown Status = iota
	// Starting is status describing unit is starting.
	Starting
	// Configuring is status describing unit is configuring.
	Configuring
	// Running is status describing unit is running.
	Running
	// Degraded is status describing unit is degraded.
	Degraded
	// Failed is status describing unit is failed. This status should
	// only be used in the case the beat should stop running as the failure
	// cannot be recovered.
	Failed
	// Stopping is status describing unit is stopping.
	Stopping
	// Stopped is status describing unit is stopped.
	Stopped
)

// StatusReporter provides a method to update current status of a unit.
type StatusReporter interface {
	// UpdateStatus updates the status of the unit.
	UpdateStatus(status Status, msg string)
}

// WithStatusReporter provides a method to set a status reporter
type WithStatusReporter interface {
	// SetStatusReporter sets the status reporter
	SetStatusReporter(reporter StatusReporter)
}
