// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package noop

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
)

// Monitor is a monitoring interface providing information about the way
// how beat is monitored
type Monitor struct {
}

// NewMonitor creates a beats monitor.
func NewMonitor() *Monitor {
	return &Monitor{}
}

// EnrichArgs enriches arguments provided to application, in order to enable
// monitoring
func (b *Monitor) EnrichArgs(_ program.Spec, _ string, args []string, _ bool) []string {
	return args
}

// Cleanup cleans up all drops.
func (b *Monitor) Cleanup(program.Spec, string) error {
	return nil
}

// Close closes the monitor.
func (b *Monitor) Close() {}

// Prepare executes steps in order for monitoring to work correctly
func (b *Monitor) Prepare(program.Spec, string, int, int) error {
	return nil
}

// LogPath describes a path where application stores logs. Empty if
// application is not monitorable
func (b *Monitor) LogPath(program.Spec, string) string {
	return ""
}

// MetricsPath describes a location where application exposes metrics
// collectable by metricbeat.
func (b *Monitor) MetricsPath(program.Spec, string) string {
	return ""
}

// MetricsPathPrefixed return metrics path prefixed with http+ prefix.
func (b *Monitor) MetricsPathPrefixed(program.Spec, string) string {
	return ""
}

// Reload reloads state based on configuration.
func (b *Monitor) Reload(cfg *config.Config) error { return nil }

// IsMonitoringEnabled returns true if monitoring is configured.
func (b *Monitor) IsMonitoringEnabled() bool { return false }

// WatchLogs return true if monitoring is configured and monitoring logs is enabled.
func (b *Monitor) WatchLogs() bool { return false }

// WatchMetrics return true if monitoring is configured and monitoring metrics is enabled.
func (b *Monitor) WatchMetrics() bool { return false }

// MonitoringNamespace returns monitoring namespace configured.
func (b *Monitor) MonitoringNamespace() string { return "default" }
