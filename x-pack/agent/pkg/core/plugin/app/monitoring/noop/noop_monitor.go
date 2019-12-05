// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package noop

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
func (b *Monitor) EnrichArgs(args []string) []string {
	return args
}

// Cleanup cleans up all drops.
func (b *Monitor) Cleanup() error {
	return nil
}

// Prepare executes steps in order for monitoring to work correctly
func (b *Monitor) Prepare(uid, gid int) error {
	return nil
}

// LogPath describes a path where application stores logs. Empty if
// application is not monitorable
func (b *Monitor) LogPath() string {
	return ""
}

// MetricsPath describes a location where application exposes metrics
// collectable by metricbeat.
func (b *Monitor) MetricsPath() string {
	return ""
}

// MetricsPathPrefixed return metrics path prefixed with http+ prefix.
func (b *Monitor) MetricsPathPrefixed() string {
	return ""
}
