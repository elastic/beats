// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package configuration

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/kibana"
	fleetreporter "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/reporter/fleet"
)

// FleetAgentConfig is the internal configuration of the agent after the enrollment is done,
// this configuration is not exposed in anyway in the elastic-agent.yml and is only internal configuration.
type FleetAgentConfig struct {
	Enabled      bool                  `config:"enabled" yaml:"enabled"`
	AccessAPIKey string                `config:"access_api_key" yaml:"access_api_key"`
	Kibana       *kibana.Config        `config:"kibana" yaml:"kibana"`
	Reporting    *fleetreporter.Config `config:"reporting" yaml:"reporting"`
	Info         *AgentInfo            `config:"agent" yaml:"agent"`
}

// Valid validates the required fields for accessing the API.
func (e *FleetAgentConfig) Valid() error {
	if e.Enabled {
		if len(e.AccessAPIKey) == 0 {
			return errors.New("empty access token", errors.TypeConfig)
		}

		if e.Kibana == nil || len(e.Kibana.Host) == 0 {
			return errors.New("missing Kibana host configuration", errors.TypeConfig)
		}
	}

	return nil
}

// DefaultFleetAgentConfig creates a default configuration for fleet.
func DefaultFleetAgentConfig() *FleetAgentConfig {
	return &FleetAgentConfig{
		Enabled:   false,
		Kibana:    kibana.DefaultClientConfig(),
		Reporting: fleetreporter.DefaultConfig(),
		Info:      &AgentInfo{},
	}
}
