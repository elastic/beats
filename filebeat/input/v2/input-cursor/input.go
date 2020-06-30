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

package cursor

import (
	"fmt"
	"time"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
)

// Input interface for cursor based inputs. This interface must be implemented
// by inputs that with to use the InputManager in order to implement a stateful
// input that can store state between restarts.
type Input interface {
	Name() string

	// Test checks the configuaration and runs additional checks if the Input can
	// actually collect data for the given configuration (e.g. check if host/port or files are
	// accessible).
	// The input manager will call Test per configured source.
	Test(Source, input.TestContext) error

	// Run starts the data collection. Run must return an error only if the
	// error is fatal making it impossible for the input to recover.
	// The input run a go-routine can call Run per configured Source.
	Run(input.Context, Source, Cursor, Publisher) error
}

// managedInput implements the v2.Input interface, integrating cursor Inputs
// with the v2 input API.
// The managedInput starts go-routines per configured source.
// If a Run returns the error is 'remembered', but active data collecting
// continues. Only after all Run calls have returned will the managedInput be
// done.
type managedInput struct {
	manager      *InputManager
	userID       string
	sources      []Source
	input        Input
	cleanTimeout time.Duration
}

// Name is required to implement the v2.Input interface
func (inp *managedInput) Name() string { return inp.input.Name() }

// Test runs the Test method for each configured source.
func (inp *managedInput) Test(ctx input.TestContext) error {
	panic("TODO: implement me")
}

// Run creates a go-routine per source, waiting until all go-routines have
// returned, either by error, or by shutdown signal.
// If an input panics, we create an error value with stack trace to report the
// issue, but not crash the whole process.
func (inp *managedInput) Run(
	ctx input.Context,
	pipeline beat.PipelineConnector,
) (err error) {
	panic("TODO: implement me")
}

func (inp *managedInput) createSourceID(s Source) string {
	if inp.userID != "" {
		return fmt.Sprintf("%v::%v::%v", inp.manager.Type, inp.userID, s.Name())
	}
	return fmt.Sprintf("%v::%v", inp.manager.Type, s.Name())
}
