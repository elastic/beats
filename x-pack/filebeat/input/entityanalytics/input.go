// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package entityanalytics

import (
	"fmt"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/unison"

	// For provider registration.
	_ "github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/activedirectory"
	_ "github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread"
	_ "github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/okta"
)

// Name of this input.
const Name = "entity-analytics"

func Plugin(logger *logp.Logger) v2.Plugin {
	return v2.Plugin{
		Name:      Name,
		Stability: feature.Experimental,
		Info:      "Identity Provider for Entity Analytics",
		Doc:       "Collect identity assets for Entity Analytics",
		Manager: &manager{
			logger: logger,
		},
	}
}

// manager implements the v2.InputManager interface.
type manager struct {
	logger   *logp.Logger
	provider provider.Provider
}

// Init is not used for this input. It is called before Create and no provider
// has been configured yet.
func (m *manager) Init(grp unison.Group) error {
	return nil
}

// Create will unpack the provided configuration and set up the identity provider
// for this input.
func (m *manager) Create(cfg *config.C) (v2.Input, error) {
	var c conf
	if err := cfg.Unpack(&c); err != nil {
		return nil, fmt.Errorf("unable to unpack %s input config: %w", Name, err)
	}

	factoryFn, err := provider.Get(c.Provider)
	if err != nil {
		return nil, fmt.Errorf("unable to create %s input: %w", Name, err)
	}

	m.provider, err = factoryFn(m.logger)
	if err != nil {
		return nil, fmt.Errorf("unable to create %s input provider: %w", Name, err)
	}

	return m.provider.Create(cfg)
}
