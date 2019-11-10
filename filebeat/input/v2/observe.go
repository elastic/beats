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

package v2

// StatusObserver is used to report a standardized set of state change events
// of an input.
type StatusObserver interface {
	RunnerObserver

	// Starting indicates that the input is about to be configured and started.
	Starting()

	// Stopped reports that the input has finished the shutdown and cleanup.
	Stopped()

	// Failed indicates that the input has been stopped due to a fatal error.
	Failed(err error)
}

// RunnerObserver reports the current state of an active input instance.
type RunnerObserver interface {
	// Initialized reports that required resources are initialized, but the
	// Input is not collecting events yet.
	Initialized()

	// Active reports that the input is about to start collecting events.
	Active()

	// Failing reports that the input is experiencing temporary errors. The input
	// does not quit yet, but will attempt to retry.
	Failing(err error)

	// Stopping reports that the input is about to stop and clean up resources.
	Stopping()
}
