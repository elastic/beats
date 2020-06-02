// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitoring

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/plugin/app/monitoring/beats"
)

// Monitor is a monitoring interface providing information about the way
// how application is monitored
type Monitor interface {
	LogPath(process, pipelineID string) string
	MetricsPath(process, pipelineID string) string
	MetricsPathPrefixed(process, pipelineID string) string

	Prepare(process, pipelineID string, uid, gid int) error
	EnrichArgs(string, string, []string, bool) []string
	Cleanup(process, pipelineID string) error
	Reload(cfg *config.Config) error
	IsMonitoringEnabled() bool
	WatchLogs() bool
	WatchMetrics() bool
}

type wrappedConfig struct {
	DownloadConfig *artifact.Config `yaml:"download" config:"download"`
}

// NewMonitor creates a monitor based on a process configuration.
func NewMonitor(config *config.Config) (Monitor, error) {
	cfg := &wrappedConfig{
		DownloadConfig: artifact.DefaultConfig(),
	}

	if err := config.Unpack(&cfg); err != nil {
		return nil, err
	}
	return beats.NewMonitor(cfg.DownloadConfig), nil
}
