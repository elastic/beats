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

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/go-concert/chorus"
)

type Plugin struct {
	Name string

	Stability Stability

	Deprecated bool

	Doc string

	// TODO: config schema info for config validation

	// Configure configures an Input if possible. Returns an error if the
	// configuration is false.
	// The input return must not attempt to run or create any kind of connection yet.
	// The input is only supposed to be a types place holder for the untyped
	// configuration.
	Configure func(log *logp.Logger, config *common.Config) (Input, error)
}

// Input defines the functionality all input plugins must provide.
type Input interface {
	InputTester
	RunnerFactory
}

// InputTester provides functionality for testing aa feature its configuration without
// having to run it for a long time.
// The test is supposed to not take too much time. Packages running a tester
// might run the test with a pre-configured timeout.
type InputTester interface {
	TestInput(closer *chorus.Closer, log *logp.Logger) error
}

// RunnerFactory creates a runner that can be started and stopped.
// The context passed to the runner is used to signal beats shutdown.
//
// The provided observer must be used to provide report the inputs lifecycle
// state change.
type RunnerFactory interface {
	CreateRunner(ctx Context) Runner
}

// Runner must run a feature until the run is either stopped, or fails hard
// making it impossible to recover.
// Each call to Run must be fully isolated and independent of other runs. No
// global state should be shared via the Input or global variables.
// If the Runner is stopped, it is assumed that all resources are freed.
//
// NOTE: the type is defined such that it is compatible with libbeat config
// reloading and autodiscovery.
type Runner interface {
	fmt.Stringer

	Start()
	Stop()
}

// Context provides access to common resources and shutdown signaling to
// inputs. The `Closer` provided is compatible to `context.Context` and can be used
// with common IO libraries to unblock connections on shutdown.
type Context struct {
	// ID of the input, for informational purposes only. Loggers and observer will already report
	// the ID if needed.
	ID string

	// StoreAccessor allows inputs to access a resource store. The store can be used
	// for serializing state, but also for coordination such that only one input
	// collects data from a resource.
	StoreAccessor

	// Closer provides support for shutdown signaling.
	// It is compatible to context.Context, and can be used to cancel IO
	// operations during shutdown.
	Closer *chorus.Closer

	// Log provides the structured logger for the input to use
	Log *logp.Logger

	// Observer is used to signal state changes. The state is used for reporting
	// the state/healthiness to users using management/monitoring APIs.
	Status StatusObserver

	// Pipeline allows inputs to connect to the active publisher pipeline. Each
	// go-routine creating and publishing events should have it's own connection.
	Pipeline beat.PipelineConnector
}
