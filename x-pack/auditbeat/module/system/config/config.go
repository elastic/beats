// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

// Config defines the host metricset's configuration options.
type Config struct {
	ReportChanges bool `config:"report_changes"`
}

// Validate validates the host metricset config.
func (c *Config) Validate() error {
	return nil
}

// NewDefaultConfig returns a default configuration for this module.
func NewDefaultConfig() Config {
	return Config{
		ReportChanges: true,
	}
}
