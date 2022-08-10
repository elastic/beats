// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package configuration

import (
	"path/filepath"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	monitoringCfg "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/process"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/retry"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/server"
)

// ExternalInputsPattern is a glob that matches the paths of external configuration files.
var ExternalInputsPattern = filepath.Join("inputs.d", "*.yml")

// SettingsConfig is an collection of agent settings configuration.
type SettingsConfig struct {
	DownloadConfig   *artifact.Config                `yaml:"download" config:"download" json:"download"`
	ProcessConfig    *process.Config                 `yaml:"process" config:"process" json:"process"`
	GRPC             *server.Config                  `yaml:"grpc" config:"grpc" json:"grpc"`
	RetryConfig      *retry.Config                   `yaml:"retry" config:"retry" json:"retry"`
	MonitoringConfig *monitoringCfg.MonitoringConfig `yaml:"monitoring" config:"monitoring" json:"monitoring"`
	LoggingConfig    *logger.Config                  `yaml:"logging,omitempty" config:"logging,omitempty" json:"logging,omitempty"`

	// standalone config
	Reload *ReloadConfig `config:"reload" yaml:"reload" json:"reload"`
	Path   string        `config:"path" yaml:"path" json:"path"`
}

// DefaultSettingsConfig creates a config with pre-set default values.
func DefaultSettingsConfig() *SettingsConfig {
	return &SettingsConfig{
		ProcessConfig:    process.DefaultConfig(),
		RetryConfig:      retry.DefaultConfig(),
		DownloadConfig:   artifact.DefaultConfig(),
		LoggingConfig:    logger.DefaultLoggingConfig(),
		MonitoringConfig: monitoringCfg.DefaultConfig(),
		GRPC:             server.DefaultGRPCConfig(),
		Reload:           DefaultReloadConfig(),
	}
}
