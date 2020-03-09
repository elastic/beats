// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitoring

import (
	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/app/monitoring/beats"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/app/monitoring/noop"
)

// Monitor is a monitoring interface providing information about the way
// how application is monitored
type Monitor interface {
	EnrichArgs([]string) []string
	Prepare(uid, gid int) error
	Cleanup() error
	LogPath() string
	MetricsPath() string
	MetricsPathPrefixed() string
}

// NewMonitor creates a monitor based on a process configuration.
func NewMonitor(isMonitorable bool, process, pipelineID string, downloadConfig *artifact.Config, monitorLogs, monitorMetrics bool) Monitor {
	if !isMonitorable {
		return noop.NewMonitor()
	}

	// so far we support only beats monitoring
	return beats.NewMonitor(process, pipelineID, downloadConfig, monitorLogs, monitorMetrics)
}
