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
	input "github.com/menderesk/beats/v7/filebeat/input/v2"
)

// Prospector is responsible for starting, stopping harvesters
// based on the retrieved information about the configured paths.
// It also updates the statestore with the meta data of the running harvesters.
type Prospector interface {
	// Init runs the cleanup processes before starting the prospector.
	Init(c, g ProspectorCleaner, newID func(Source) string) error
	// Run starts the event loop and handles the incoming events
	// either by starting/stopping a harvester, or updating the statestore.
	Run(input.Context, StateMetadataUpdater, HarvesterGroup)
	// Test checks if the Prospector is able to run the configuration
	// specified by the user.
	Test() error
}

// StateMetadataUpdater updates and removes the state information for a given Source.
type StateMetadataUpdater interface {
	// FindCursorMeta retrieves and unpacks the cursor metadata of an entry of the given Source.
	FindCursorMeta(s Source, v interface{}) error
	// UpdateMetadata updates the source metadata of a registry entry of a given Source.
	UpdateMetadata(s Source, v interface{}) error
	// Remove marks a state for deletion of a given Source.
	Remove(s Source) error
	// ResetCursor resets the cursor in the registry and drops previous state
	// updates that are not yet ACKed.
	ResetCursor(s Source, cur interface{}) error
}

// ProspectorCleaner cleans the state store before it starts running.
type ProspectorCleaner interface {
	// CleanIf removes an entry if the function returns true
	CleanIf(func(v Value) bool)
	// UpdateIdentifiers updates ID in the registry.
	// The function passed to UpdateIdentifiers must return an empty string if the key
	// remains the same.
	UpdateIdentifiers(func(v Value) (string, interface{}))

	// FixUpIdentifiers migrates IDs in the registry from inputs
	// that used the deprecated `.global` ID.
	FixUpIdentifiers(func(v Value) (string, interface{}))
}

// Value contains the cursor metadata.
type Value interface {
	// UnpackCursorMeta returns the cursor metadata required by the prospector.
	UnpackCursorMeta(to interface{}) error
}
