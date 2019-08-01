// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package events

import "github.com/elastic/fleet/x-pack/pkg/id"

const (
	// StateChangeRun is a name of Start program event
	StateChangeRun = "sc-run"
	// StateChangeRemove is a name of Remove program event causing beat in version to be uninstalled
	StateChangeRemove = "sc-remove"
	// StateChangeStartSidecar is a name of start program monitoring event
	StateChangeStartSidecar = "sc-sidecar-start"
	// StateChangeStopSidecar is a name of stop program monitoring event
	StateChangeStopSidecar = "sc-sidecar-stop"

	// MetaConfigKey is key used to store configuration in metadata
	MetaConfigKey = "config"
)

// StateChangeEvent is an event produced by state resolver describing
// a change needed to be applied such as starting/stopping processes or
// applying configurations.
type StateChangeEvent struct {
	Steps []Step
	id    id.ID
}

// Step is a step needed to be applied
type Step struct {
	// ID identifies kind of operation needed to be executed
	ID string
	// Version is a version of a program
	Version string
	// Process defines a process such as `filebeat`
	Process string
	// Meta contains additional data such as version, configuration or tags.
	Meta map[string]interface{}
}

// ID returns an id for the operation,
func (s *StateChangeEvent) ID() id.ID {
	return s.id
}

// NewStateChange creates a new event with predefined steps
func NewStateChange(steps []Step) (*StateChangeEvent, error) {
	id, err := id.Generate()
	if err != nil {
		return nil, err
	}
	return &StateChangeEvent{id: id, Steps: steps}, nil
}
