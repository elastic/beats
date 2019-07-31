// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

import (
	xmanagement "github.com/elastic/beats/x-pack/libbeat/management"
)

// Config for central management
type Config struct {
	Fleet     *FleetConfig                        `config:"fleet" yaml:"fleet"`
	Blacklist xmanagement.ConfigBlacklistSettings `config:"blacklist" yaml:"blacklist"`
}

// FleetConfig configuration of the fleet.
type FleetConfig struct {
	// true when enrolled
	Enabled bool `config:"enabled" yaml:"enabled"`
}

func defaultConfig() *Config {
	return &Config{
		Blacklist: xmanagement.ConfigBlacklistSettings{
			Patterns: map[string]string{
				"output": "console|file",
			},
		},
	}
}
