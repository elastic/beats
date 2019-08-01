// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package events

import (
	"github.com/elastic/fleet/x-pack/pkg/agent/program"
	"github.com/elastic/fleet/x-pack/pkg/id"
)

// ConfigChangedEvent describe a config change.
type ConfigChangedEvent struct {
	id      id.ID
	Program program.Program
}

// ID returns the id of the event.
func (c *ConfigChangedEvent) ID() id.ID {
	return c.id
}

// NewConfigChanged creates a ConfigChanged event for a specific program.
func NewConfigChanged(program program.Program) (*ConfigChangedEvent, error) {
	id, err := id.Generate()
	if err != nil {
		return nil, err
	}
	return &ConfigChangedEvent{id: id, Program: program}, nil
}
