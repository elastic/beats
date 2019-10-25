// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"fmt"
	"time"

	"github.com/elastic/beats/x-pack/agent/pkg/config"
	fleetreporter "github.com/elastic/beats/x-pack/agent/pkg/reporter/fleet"
	logreporter "github.com/elastic/beats/x-pack/agent/pkg/reporter/log"
)

// Config define the configuration of the Agent.
type Config struct {
	Management *config.Config `config:"management"`
}

func localDefaultConfig() *Config {
	return &Config{}
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
	Mode      managementMode   `config:"mode"`
	Fleet     *fleetConfig     `config:"fleet"`
	Reporting *reportingConfig `config:"reporting"`
}

type reportingConfig struct {
	LogReporting    *logreporter.Config             `config:"log"`
	FleetManagement *fleetreporter.ManagementConfig `config:"fleet"`
}

func defaultManagementConfig() *ManagementConfig {
	return &ManagementConfig{
		Mode: localMode,
		Reporting: &reportingConfig{
			LogReporting:    logreporter.DefaultLogConfig(),
			FleetManagement: fleetreporter.DefaultFleetManagementConfig(),
		},
	}
}

type localConfig struct {
	Reload *reloadConfig `config:"reload"`
	Path   string        `config:"path"`
}

type reloadConfig struct {
	Enabled bool          `config:"enabled"`
	Period  time.Duration `config:"period"`
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
		Reload: &reloadConfig{
			Enabled: true,
			Period:  10 * time.Second,
		},
	}
}
