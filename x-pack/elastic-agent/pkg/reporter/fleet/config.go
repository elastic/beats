// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

// Config is a configuration describing fleet connected parts
type Config struct {
	Threshold               int `yaml:"threshold" config:"threshold" validate:"min=1"`
	ReportingCheckFrequency int `yaml:"check_frequency_sec" config:"check_frequency_sec" validate:"min=1"`
}

// DefaultConfig initiates FleetManagementConfig with default values
func DefaultConfig() *Config {
	return &Config{
		Threshold:               10000,
		ReportingCheckFrequency: 30,
	}
}
