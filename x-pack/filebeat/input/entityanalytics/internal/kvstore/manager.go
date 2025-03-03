// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"
)

// Manager is used to create, manage, and coordinate inputs which use a key/value
// store for their persistent state.
type Manager struct {
	// Logger for writing log messages.
	Logger *logp.Logger

	// Type must contain the name of the input type.
	Type string

	// Configure returns a configured Input instance and a slice of Sources
	// that will be used to collect events.
	Configure func(cfg *config.C) (Input, error)
}

// managerConfig contains parameters needed to configure the Manager.
type managerConfig struct {
	ID string `config:"id" validate:"required"`
}

// Init initializes any required resources. It is currently a no-op.
func (m *Manager) Init(grp unison.Group) error {
	return nil
}

// Create makes a new v2.Input using the provided config.C which will be
// used in the Manager's Configure function.
func (m *Manager) Create(c *config.C) (v2.Input, error) {
	inp, err := m.Configure(c)
	if err != nil {
		return nil, err
	}

	settings := managerConfig{}
	if err = c.Unpack(&settings); err != nil {
		return nil, err
	}

	return &input{
		id:           settings.ID,
		manager:      m,
		managedInput: inp,
	}, nil
}
