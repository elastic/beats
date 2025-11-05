// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqdcli"
)

const (
	// Interval for collecting osqueryd health metrics
	osquerydHealthCheckInterval = 30 * time.Second

	// Memory threshold in bytes (500MB)
	osquerydMemoryThresholdBytes = 500 * 1024 * 1024
)

// osquerydMetrics holds monitoring metrics for osqueryd process
// Follows beat monitoring naming conventions with _gauge suffix for gauges
type osquerydMetrics struct {
	pid            *monitoring.Uint   // pid_gauge - Process ID (gauge, can change on restart)
	memoryResident *monitoring.Uint   // memory.rss_gauge - Resident Set Size in bytes (gauge)
	memoryVirtual  *monitoring.Uint   // memory.vms_gauge - Virtual Memory Size in bytes (gauge)
	cpuUser        *monitoring.Uint   // cpu.user_total - Cumulative CPU time in user space (milliseconds, counter)
	cpuSystem      *monitoring.Uint   // cpu.system_total - Cumulative CPU time in kernel space (milliseconds, counter)
	threads        *monitoring.Uint   // threads_gauge - Number of threads (gauge)
	state          *monitoring.String // state - Process state (R, S, D, Z, etc.)
	startTime      *monitoring.Uint   // start_time - Unix timestamp when process started
	diskReadBytes  *monitoring.Uint   // disk.read_bytes_total - Cumulative bytes read from disk (counter)
	diskWriteBytes *monitoring.Uint   // disk.write_bytes_total - Cumulative bytes written to disk (counter)
	uptime         *monitoring.Uint   // uptime_seconds - Seconds the process has been running (gauge)

	log *logp.Logger
}

// processMetrics represents osqueryd process metrics from osquery
type processMetrics struct {
	PID              int64  `json:"pid,string"`
	ResidentSize     int64  `json:"resident_size,string"`
	TotalSize        int64  `json:"total_size,string"`
	UserTime         int64  `json:"user_time,string"`
	SystemTime       int64  `json:"system_time,string"`
	Threads          int64  `json:"threads,string"`
	State            string `json:"state"`
	StartTime        int64  `json:"start_time,string"`
	DiskBytesRead    int64  `json:"disk_bytes_read,string"`
	DiskBytesWritten int64  `json:"disk_bytes_written,string"`
}

// newOsquerydMetrics creates and registers osqueryd health metrics
// Follows beat conventions: _gauge suffix for gauges, _total for cumulative counters
func newOsquerydMetrics(registry *monitoring.Registry, log *logp.Logger) *osquerydMetrics {
	// Create a sub-registry for osqueryd metrics
	osqdReg := registry.NewRegistry("osqueryd")

	return &osquerydMetrics{
		pid:            monitoring.NewUint(osqdReg, "pid_gauge"),
		memoryResident: monitoring.NewUint(osqdReg, "memory.rss_gauge"),
		memoryVirtual:  monitoring.NewUint(osqdReg, "memory.vms_gauge"),
		cpuUser:        monitoring.NewUint(osqdReg, "cpu.user_total"),
		cpuSystem:      monitoring.NewUint(osqdReg, "cpu.system_total"),
		threads:        monitoring.NewUint(osqdReg, "threads_gauge"),
		state:          monitoring.NewString(osqdReg, "state"),
		startTime:      monitoring.NewUint(osqdReg, "start_time"),
		diskReadBytes:  monitoring.NewUint(osqdReg, "disk.read_bytes_total"),
		diskWriteBytes: monitoring.NewUint(osqdReg, "disk.write_bytes_total"),
		uptime:         monitoring.NewUint(osqdReg, "uptime_seconds"),
		log:            log,
	}
}

// update queries osqueryd process metrics and updates the monitoring registry
func (m *osquerydMetrics) update(ctx context.Context, osq osqd.Runner) error {
	metrics, err := m.queryOsquerydMetrics(ctx, osq)
	if err != nil {
		return fmt.Errorf("failed to query osqueryd metrics: %w", err)
	}

	// Update monitoring metrics
	m.pid.Set(uint64(metrics.PID))
	m.memoryResident.Set(uint64(metrics.ResidentSize))
	m.memoryVirtual.Set(uint64(metrics.TotalSize))
	m.cpuUser.Set(uint64(metrics.UserTime))
	m.cpuSystem.Set(uint64(metrics.SystemTime))
	m.threads.Set(uint64(metrics.Threads))
	m.state.Set(metrics.State)
	m.startTime.Set(uint64(metrics.StartTime))
	m.diskReadBytes.Set(uint64(metrics.DiskBytesRead))
	m.diskWriteBytes.Set(uint64(metrics.DiskBytesWritten))

	// Calculate and set uptime if start_time is available
	if metrics.StartTime > 0 {
		uptime := time.Now().Unix() - metrics.StartTime
		if uptime > 0 {
			m.uptime.Set(uint64(uptime))
		}
	}

	m.log.Debugf("Updated osqueryd metrics: pid=%d, memory=%d bytes, state=%s",
		metrics.PID, metrics.ResidentSize, metrics.State)

	return nil
}

// queryOsquerydMetrics queries osqueryd process metrics using osquery
func (m *osquerydMetrics) queryOsquerydMetrics(ctx context.Context, osq osqd.Runner) (*processMetrics, error) {
	// Query for osqueryd process metrics by joining with osquery_info
	// to get the current osqueryd process PID
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
		p.disk_bytes_written
	FROM processes p
	JOIN osquery_info o ON p.pid = o.pid`

	// Create osquery client using the socket path from the runner
	client := osqdcli.New(osq.SocketPath())
	defer client.Close()

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

	// Parse the response
	var metrics processMetrics
	data, err := json.Marshal(resp[0])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal process metrics: %w", err)
	}

	return &metrics, nil
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
	if m.startTime.Get() > 0 {
		uptime := time.Now().Unix() - int64(m.startTime.Get())
		if uptime < 60 { // Less than 1 minute uptime
			warnings = append(warnings, fmt.Sprintf("process recently started (uptime: %ds)", uptime))
		}
	}

	return warnings
}

// monitorOsquerydHealth periodically collects osqueryd metrics and reports issues
func monitorOsquerydHealth(ctx context.Context, osq osqd.Runner, metrics *osquerydMetrics, log *logp.Logger) {
	ticker := time.NewTicker(osquerydHealthCheckInterval)
	defer ticker.Stop()

	log.Info("Starting osqueryd health monitoring")

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping osqueryd health monitoring")
			return
		case <-ticker.C:
			if err := metrics.update(ctx, osq); err != nil {
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
