// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqdcli"
)

const (
	// Interval for collecting osqueryd health metrics
	osquerydHealthCheckInterval = 30 * time.Second

	// Memory threshold in bytes (500MB)
	osquerydMemoryThresholdBytes = 500 * 1024 * 1024
)

// osquerydMetrics holds monitoring metrics for osqueryd process
// Follows beat monitoring naming conventions
type osquerydMetrics struct {
	pid            *monitoring.Uint   // pid - Process ID (gauge, can change on restart)
	memoryResident *monitoring.Uint   // memory.rss.bytes - Resident Set Size in bytes (gauge)
	memoryVirtual  *monitoring.Uint   // memory.virtual.bytes - Virtual Memory Size in bytes (gauge)
	cpuUserTime    *monitoring.Uint   // cpu.user.time.ms - Cumulative CPU time in user space in milliseconds (counter)
	cpuSystemTime  *monitoring.Uint   // cpu.system.time.ms - Cumulative CPU time in kernel space in milliseconds (counter)
	threads        *monitoring.Uint   // threads - Number of threads (gauge)
	state          *monitoring.String // state - Process state (R, S, D, Z, etc.)
	startTime      uint64             // internal: Unix timestamp when process started (not exposed)
	diskReadBytes  *monitoring.Uint   // disk.read_bytes_total - Cumulative bytes read from disk (counter)
	diskWriteBytes *monitoring.Uint   // disk.write_bytes_total - Cumulative bytes written to disk (counter)
	uptime         *monitoring.Uint   // uptime.ms - Milliseconds the process has been running (gauge)
	version        *monitoring.String // version - osqueryd version string

	log *logp.Logger
}

// getUint64 extracts an int64 from map[string]any, handling both string and numeric values
func getUint64(m map[string]any, key string) uint64 {
	val, ok := m[key]
	if !ok {
		return 0
	}

	switch v := val.(type) {
	case int64:
		return uint64(v)
	case float64:
		return uint64(v)
	case int:
		return uint64(v)
	case string:
		num, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return 0
		}
		return num
	}
	return 0
}

// getString extracts a string from map[string]any
func getString(m map[string]any, key string) string {
	val, ok := m[key]
	if !ok {
		return ""
	}
	str, _ := val.(string)
	return str
}

// newOsquerydMetrics creates and registers osqueryd health metrics
// Follows beat conventions: _total suffix for cumulative counters
func newOsquerydMetrics(registry *monitoring.Registry, log *logp.Logger) *osquerydMetrics {
	// Create a sub-registry for osqueryd metrics
	osqdReg := registry.GetOrCreateRegistry("osqueryd")

	return &osquerydMetrics{
		pid:            monitoring.NewUint(osqdReg, "pid"),
		memoryResident: monitoring.NewUint(osqdReg, "memory.rss.bytes"),
		memoryVirtual:  monitoring.NewUint(osqdReg, "memory.virtual.bytes"),
		cpuUserTime:    monitoring.NewUint(osqdReg, "cpu.user.time.ms"),
		cpuSystemTime:  monitoring.NewUint(osqdReg, "cpu.system.time.ms"),
		threads:        monitoring.NewUint(osqdReg, "threads"),
		state:          monitoring.NewString(osqdReg, "state"),
		diskReadBytes:  monitoring.NewUint(osqdReg, "disk.read_bytes_total"),
		diskWriteBytes: monitoring.NewUint(osqdReg, "disk.write_bytes_total"),
		uptime:         monitoring.NewUint(osqdReg, "uptime.ms"),
		version:        monitoring.NewString(osqdReg, "version"),
		log:            log,
	}
}

// update queries osqueryd process metrics and updates the monitoring registry
func (m *osquerydMetrics) update(ctx context.Context, client *osqdcli.Client) error {
	metrics, err := m.queryOsquerydMetrics(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to query osqueryd metrics: %w", err)
	}

	m.pid.Set(getUint64(metrics, "pid"))
	m.memoryResident.Set(getUint64(metrics, "resident_size"))
	m.memoryVirtual.Set(getUint64(metrics, "total_size"))
	m.cpuUserTime.Set(getUint64(metrics, "user_time"))
	m.cpuSystemTime.Set(getUint64(metrics, "system_time"))
	m.threads.Set(getUint64(metrics, "threads"))
	m.state.Set(getString(metrics, "state"))
	m.startTime = getUint64(metrics, "start_time")
	m.diskReadBytes.Set(getUint64(metrics, "disk_bytes_read"))
	m.diskWriteBytes.Set(getUint64(metrics, "disk_bytes_written"))
	m.version.Set(getString(metrics, "version"))

	if m.startTime > 0 {
		uptime := time.Now().Unix() - int64(m.startTime)
		if uptime > 0 {
			m.uptime.Set(uint64(uptime * 1000))
		}
	}
	return nil
}

// queryOsquerydMetrics queries osqueryd process metrics using osquery
func (m *osquerydMetrics) queryOsquerydMetrics(ctx context.Context, client *osqdcli.Client) (map[string]any, error) {
	// Query for osqueryd process metrics by joining with osquery_info
	// to get the current osqueryd process PID and version
	query := `SELECT 
		p.pid, 
		p.resident_size, 
		p.total_size, 
		p.user_time, 
		p.system_time, 
		p.threads, 
		p.state,
		p.start_time,
		p.disk_bytes_read,
		p.disk_bytes_written,
		o.version
	FROM processes p
	JOIN osquery_info o ON p.pid = o.pid`

	// Execute query with timeout
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.Query(queryCtx, query, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	if len(resp) == 0 {
		return nil, fmt.Errorf("no osqueryd process found in processes table")
	}

	if len(resp) > 1 {
		m.log.Warnf("Expected 1 row for osqueryd metrics but got %d, using first row", len(resp))
	}
	return resp[0], nil
}

// checkHealth examines the metrics and returns warnings about potential health issues
func (m *osquerydMetrics) checkHealth() []string {
	var warnings []string

	// Check memory usage
	if m.memoryResident.Get() > osquerydMemoryThresholdBytes {
		warnings = append(warnings, fmt.Sprintf("high memory usage: %d bytes", m.memoryResident.Get()))
	}

	// Check process state
	switch m.state.Get() {
	case "Z":
		warnings = append(warnings, "process in zombie state")
	case "D":
		warnings = append(warnings, "process in uninterruptible sleep (disk I/O issue)")
	}

	// Check if process is very young (possible crash/restart loop)
	if m.startTime > 0 {
		uptime := time.Now().Unix() - int64(m.startTime)
		if uptime < 60 { // Less than 1 minute uptime
			warnings = append(warnings, fmt.Sprintf("process recently started (uptime: %ds)", uptime))
		}
	}

	return warnings
}

// monitorOsquerydHealth periodically collects osqueryd metrics and reports issues
func monitorOsquerydHealth(ctx context.Context, client *osqdcli.Client, metrics *osquerydMetrics, log *logp.Logger) {
	ticker := time.NewTicker(osquerydHealthCheckInterval)
	defer ticker.Stop()

	log.Info("Starting osqueryd health monitoring")

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping osqueryd health monitoring")
			return
		case <-ticker.C:
			if err := metrics.update(ctx, client); err != nil {
				log.Warnf("Failed to update osqueryd metrics: %v", err)
				continue
			}

			// Check for health issues
			warnings := metrics.checkHealth()
			for _, warning := range warnings {
				log.Warnf("osqueryd health issue: %s", warning)
			}
		}
	}
}
