// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitoring

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/beats"
	monitoringConfig "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/config"
)

// Monitor is a monitoring interface providing information about the way
// how application is monitored
type Monitor interface {
	LogPath(spec program.Spec, pipelineID string) string
	MetricsPath(spec program.Spec, pipelineID string) string
	MetricsPathPrefixed(spec program.Spec, pipelineID string) string

	Prepare(spec program.Spec, pipelineID string, uid, gid int) error
	EnrichArgs(spec program.Spec, pipelineID string, args []string, isSidecar bool) []string
	Cleanup(spec program.Spec, pipelineID string) error
	Reload(cfg *config.Config) error
	IsMonitoringEnabled() bool
	WatchLogs() bool
	WatchMetrics() bool
	Close()
}

// TODO: changeme
type wrappedConfig struct {
	DownloadConfig   *artifact.Config                   `yaml:"agent.download" config:"agent.download"`
	MonitoringConfig *monitoringConfig.MonitoringConfig `config:"agent.monitoring" yaml:"agent.monitoring"`
}

// NewMonitor creates a monitor based on a process configuration.
func NewMonitor(cfg *configuration.SettingsConfig) (Monitor, error) {
	return beats.NewMonitor(cfg.DownloadConfig, cfg.MonitoringConfig), nil
}
