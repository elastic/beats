// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/kibana"
	fleetreporter "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/reporter/fleet"
)

// Config define the configuration of the Agent.
type Config struct {
	Management *config.Config `config:"management"`
}

func localDefaultConfig() *Config {
	localModeCfg, _ := config.NewConfigFrom(map[string]interface{}{
		"mode": "local",
	})

	return &Config{
		Management: localModeCfg,
	}
}

type managementMode int

// Define the supported mode of management.
const (
	localMode managementMode = iota + 1
	fleetMode
)

var managementModeMap = map[string]managementMode{
	"local": localMode,
	"fleet": fleetMode,
}

func (m *managementMode) Unpack(v string) error {
	mgt, ok := managementModeMap[v]
	if !ok {
		return fmt.Errorf(
			"unknown management mode, received '%s' and valid values are local or fleet",
			v,
		)
	}
	*m = mgt
	return nil
}

// ManagementConfig defines the options for the running of the beats.
type ManagementConfig struct {
	Mode managementMode `config:"mode"`
}

func defaultManagementConfig() *ManagementConfig {
	return &ManagementConfig{
		Mode: localMode,
	}
}

type localConfig struct {
	Management *localManagementConfig `config:"management" yaml:"management"`
}

type localManagementConfig struct {
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
		Management: &localManagementConfig{
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
	if len(e.AccessAPIKey) == 0 {
		return errors.New("empty access token", errors.TypeConfig)
	}

	if e.Kibana == nil || len(e.Kibana.Host) == 0 {
		return errors.New("missing Kibana host configuration", errors.TypeConfig)
	}

	return nil
}

func defaultFleetAgentConfig() *FleetAgentConfig {
	return &FleetAgentConfig{
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
