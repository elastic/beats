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
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// Input is a configured input object that can be used to probe or start
// a actual data processing.
//
// Input is a struct instead of an input. The Name and Run fields are
// mandatory, all other fields are optional. Optional fields can communicate
// additional functionality that Beats can make use of.
type Input struct {
	// Name of the input instance (mandatory).
	Name string

	// Run function to start the actual input process (mandatory).
	// The Context is used to signal shutdown.
	Run func(Context, beat.PipelineConnector) error

	// Test function to check if the input can actually run (mandatory).
	Test func(TestContext) error
}

// Context provides the Input Run function with common environmental
// information and services.
type Context struct {
	ID          string
	Agent       beat.Info
	Logger      *logp.Logger
	Status      RunnerObserver
	Metadata    *common.MapStrPointer // XXX: from Autodiscovery.
	Cancelation Canceler
}

// TestContext provides the Input Test function with common environmental
// information and services.
type TestContext struct {
	Agent       beat.Info
	Logger      *logp.Logger
	Cancelation Canceler
}

// Canceler is used to provide shutdown handling to the Context.
type Canceler interface {
	Done() <-chan struct{}
	Err() error
}
