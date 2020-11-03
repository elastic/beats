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

package input_logfile

import (
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/statestore"
)

// Prospector is responsible for starting, stopping harvesters
// based on the retrieved information about the configured paths.
// It also updates the statestore with the meta data of the running harvesters.
type Prospector interface {
	// Run starts the event loop and handles the incoming events
	// either by starting/stopping a harvester, or updating the statestore.
	Run(input.Context, *statestore.Store, HarvesterGroup)
	// Test checks if the Prospector is able to run the configuration
	// specified by the user.
	Test() error
}
