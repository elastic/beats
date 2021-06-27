// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

const defaultPort = 6791

// MonitoringConfig describes a configuration of a monitoring
type MonitoringConfig struct {
	Enabled        bool                  `yaml:"enabled" config:"enabled"`
	MonitorLogs    bool                  `yaml:"logs" config:"logs"`
	MonitorMetrics bool                  `yaml:"metrics" config:"metrics"`
	HTTP           *MonitoringHTTPConfig `yaml:"http" config:"http"`
}

// MonitoringHTTPConfig is a config defining HTTP endpoint published by agent
// for other processes to watch its metrics.
// Processes are only exposed when HTTP is enabled.
type MonitoringHTTPConfig struct {
	Enabled bool   `yaml:"enabled" config:"enabled"`
	Host    string `yaml:"host" config:"host"`
	Port    int    `yaml:"port" config:"port" validate:"min=0,max=65535,nonzero"`
}

// DefaultConfig creates a config with pre-set default values.
func DefaultConfig() *MonitoringConfig {
	return &MonitoringConfig{
		Enabled:        true,
		MonitorLogs:    true,
		MonitorMetrics: true,
		HTTP: &MonitoringHTTPConfig{
			Enabled: false,
			Port:    defaultPort,
		},
	}
}
