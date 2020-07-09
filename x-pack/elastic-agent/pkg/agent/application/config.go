// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/kibana"
	fleetreporter "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/reporter/fleet"
)

type localConfig struct {
	Fleet    *FleetAgentConfig    `config:"fleet"`
	Settings *localSettingsConfig `config:"agent" yaml:"agent"`
}

type localSettingsConfig struct {
	Reload *reloadConfig `config:"reload" yaml:"reload"`
	Path   string        `config:"path" yaml:"path"`
}

type reloadConfig struct {
	Enabled bool          `config:"enabled" yaml:"enabled"`
	Period  time.Duration `config:"period" yaml:"period"`
}

func (r *reloadConfig) Validate() error {
	if r.Enabled {
		if r.Period <= 0 {
			return ErrInvalidPeriod
		}
	}
	return nil
}

func localConfigDefault() *localConfig {
	return &localConfig{
		Fleet: defaultFleetAgentConfig(),
		Settings: &localSettingsConfig{
			Reload: &reloadConfig{
				Enabled: true,
				Period:  10 * time.Second,
			},
		},
	}
}

// FleetAgentConfig is the internal configuration of the agent after the enrollment is done,
// this configuration is not exposed in anyway in the elastic-agent.yml and is only internal configuration.
type FleetAgentConfig struct {
	Enabled      bool           `config:"enabled" yaml:"enabled"`
	AccessAPIKey string         `config:"access_api_key" yaml:"access_api_key"`
	Kibana       *kibana.Config `config:"kibana" yaml:"kibana"`
	Reporting    *LogReporting  `config:"reporting" yaml:"reporting"`
	Info         *AgentInfo     `config:"agent" yaml:"agent"`
}

// AgentInfo is a set of agent information.
type AgentInfo struct {
	ID string `json:"id" yaml:"id" config:"id"`
}

// LogReporting define the fleet options for log reporting.
type LogReporting struct {
	Fleet *fleetreporter.ManagementConfig `config:"fleet" yaml:"fleet"`
}

// Validate validates the required fields for accessing the API.
func (e *FleetAgentConfig) Validate() error {
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

func defaultFleetAgentConfig() *FleetAgentConfig {
	return &FleetAgentConfig{
		Enabled: false,
		Reporting: &LogReporting{
			Fleet: fleetreporter.DefaultFleetManagementConfig(),
		},
		Info: &AgentInfo{},
	}
}

func createFleetConfigFromEnroll(agentID string, accessAPIKey string, kbn *kibana.Config) (*FleetAgentConfig, error) {
	cfg := defaultFleetAgentConfig()
	cfg.AccessAPIKey = accessAPIKey
	cfg.Kibana = kbn
	cfg.Info.ID = agentID

	if err := cfg.Validate(); err != nil {
		return nil, errors.New(err, "invalid enrollment options", errors.TypeConfig)
	}
	return cfg, nil
}
