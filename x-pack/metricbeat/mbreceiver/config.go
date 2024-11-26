// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mbreceiver

import "fmt"

// Config is config settings for metricbeat receiver.  The structure of
// which is the same as the metricbeat.yml configuration file.
type Config struct {
	Beatconfig map[string]interface{} `mapstructure:",remain"`
}

// Validate checks if the configuration in valid
func (cfg *Config) Validate() error {
	if len(cfg.Beatconfig) == 0 {
		return fmt.Errorf("Configuration is required")
	}
	_, prs := cfg.Beatconfig["metricbeat"]
	if !prs {
		return fmt.Errorf("Configuration key 'metricbeat' is required")
	}
	return nil
}
