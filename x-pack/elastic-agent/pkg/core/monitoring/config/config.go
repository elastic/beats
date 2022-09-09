// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

const defaultPort = 6791
const defaultNamespace = "default"

// MonitoringConfig describes a configuration of a monitoring
type MonitoringConfig struct {
	Enabled        bool                  `yaml:"enabled" config:"enabled"`
	MonitorLogs    bool                  `yaml:"logs" config:"logs"`
	MonitorMetrics bool                  `yaml:"metrics" config:"metrics"`
	LogMetrics     bool                  `yaml:"-" config:"-"`
	HTTP           *MonitoringHTTPConfig `yaml:"http" config:"http"`
	Namespace      string                `yaml:"namespace" config:"namespace"`
	Pprof          *PprofConfig          `yaml:"pprof" config:"pprof"`
}

// MonitoringHTTPConfig is a config defining HTTP endpoint published by agent
// for other processes to watch its metrics.
// Processes are only exposed when HTTP is enabled.
type MonitoringHTTPConfig struct {
	Enabled bool          `yaml:"enabled" config:"enabled"`
	Host    string        `yaml:"host" config:"host"`
	Port    int           `yaml:"port" config:"port" validate:"min=0,max=65535,nonzero"`
	Buffer  *BufferConfig `yaml:"buffer" config:"buffer"`
}

// PprofConfig is a struct for the pprof enablement flag.
// It is a nil struct by default to allow the agent to use the a value that the user has injected into fleet.yml as the source of truth that is passed to beats
// TODO get this value from Kibana?
type PprofConfig struct {
	Enabled bool `yaml:"enabled" config:"enabled"`
}

// BufferConfig is a struct for for the metrics buffer endpoint
type BufferConfig struct {
	Enabled bool `yaml:"enabled" config:"enabled"`
}

// DefaultConfig creates a config with pre-set default values.
func DefaultConfig() *MonitoringConfig {
	return &MonitoringConfig{
		Enabled:        true,
		MonitorLogs:    true,
		MonitorMetrics: true,
		LogMetrics:     true,
		HTTP: &MonitoringHTTPConfig{
			Enabled: false,
			Port:    defaultPort,
		},
		Namespace: defaultNamespace,
	}
}
