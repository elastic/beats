// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitoring

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/beats"
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
	MonitoringNamespace() string
	WatchLogs() bool
	WatchMetrics() bool
	Close()
}

// NewMonitor creates beats a monitor based on a process configuration.
func NewMonitor(cfg *configuration.SettingsConfig) (Monitor, error) {
	logMetrics := true
	if cfg.LoggingConfig != nil {
		logMetrics = cfg.LoggingConfig.Metrics.Enabled
	}
	return beats.NewMonitor(cfg.DownloadConfig, cfg.MonitoringConfig, logMetrics), nil
}
