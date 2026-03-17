// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

const osqueryScheduleProfileQueryPrefix = `
SELECT
  name,
  executions,
  last_executed,
  output_size,
  wall_time_ms,
  last_wall_time_ms,
  user_time,
  last_user_time,
  system_time,
  last_system_time,
  average_memory,
  last_memory
FROM osquery_schedule
WHERE name = '`

const osqueryScheduleProfileQuerySuffix = `'`

const osqueryScheduleProfilesDiagnosticsQuery = `
SELECT
  name,
  executions,
  last_executed,
  output_size,
  wall_time_ms,
  last_wall_time_ms,
  user_time,
  last_user_time,
  system_time,
  last_system_time,
  average_memory,
  last_memory
FROM osquery_schedule
`

type runtimeSnapshot struct {
	pid          int64
	residentSize int64
	userTimeMS   int64
	systemTimeMS int64
	fds          int64
}

type scheduleTotals struct {
	executions int64
	wallMS     int64
	userMS     int64
	systemMS   int64
	outputSize int64
}

type queryProfiler struct {
	log           *logp.Logger
	mx            sync.Mutex
	scheduleState map[string]scheduleTotals
}

func newQueryProfiler(log *logp.Logger) *queryProfiler {
	return &queryProfiler{
		log:           log,
		scheduleState: make(map[string]scheduleTotals),
	}
}

func (p *queryProfiler) profileScheduledQuery(ctx context.Context, qe queryExecutor, queryName string) (map[string]interface{}, error) {
	escapedName := strings.ReplaceAll(queryName, "'", "''")
	query := osqueryScheduleProfileQueryPrefix + escapedName + osqueryScheduleProfileQuerySuffix
	rows, err := qe.Query(ctx, query, 10*time.Second)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no osquery_schedule metrics for query %q", queryName)
	}

	row := rows[0]
	executions := toInt64(row["executions"])
	lastExecuted := toInt64(row["last_executed"])
	outputSizeTotal := toInt64(row["output_size"])

	wallTotal := toInt64(row["wall_time_ms"])
	lastWall := toInt64(row["last_wall_time_ms"])

	userTotal := toInt64(row["user_time"])
	lastUser := toInt64(row["last_user_time"])

	systemTotal := toInt64(row["system_time"])
	lastSystem := toInt64(row["last_system_time"])

	lastMemory := toInt64(row["last_memory"])
	if lastMemory == 0 {
		lastMemory = toInt64(row["average_memory"])
	}

	p.mx.Lock()
	prev := p.scheduleState[queryName]
	p.scheduleState[queryName] = scheduleTotals{
		executions: executions,
		wallMS:     wallTotal,
		userMS:     userTotal,
		systemMS:   systemTotal,
		outputSize: outputSizeTotal,
	}
	p.mx.Unlock()

	execDelta := executions - prev.executions
	if execDelta <= 0 {
		// Execution count did not increase (e.g. osqueryd restarted). Treat as single run using current totals.
		execDelta = 1
		prev = scheduleTotals{}
	}

	// Prefer osquery "last_*" metrics when present. Fall back to derived per-run values.
	wallPerExec := lastWall
	if wallPerExec <= 0 {
		wallPerExec = (wallTotal - prev.wallMS) / execDelta
	}
	if wallPerExec < 0 {
		wallPerExec = 0
	}

	userPerExec := lastUser
	if userPerExec <= 0 {
		userPerExec = (userTotal - prev.userMS) / execDelta
	}
	if userPerExec < 0 {
		userPerExec = 0
	}

	systemPerExec := lastSystem
	if systemPerExec <= 0 {
		systemPerExec = (systemTotal - prev.systemMS) / execDelta
	}
	if systemPerExec < 0 {
		systemPerExec = 0
	}

	outputPerExec := (outputSizeTotal - prev.outputSize) / execDelta
	if outputPerExec < 0 {
		outputPerExec = 0
	}

	cpuMS := userPerExec + systemPerExec
	profile := map[string]interface{}{
		"source":         "scheduled",
		"query_name":     queryName,
		"utilization":    utilizationFromMillis(cpuMS, wallPerExec),
		"duration":       millisToSeconds(wallPerExec),
		"memory":         lastMemory,
		"user_time":      millisToSeconds(userPerExec),
		"system_time":    millisToSeconds(systemPerExec),
		"cpu_time":       millisToSeconds(cpuMS),
		"output_size":    outputPerExec,
		"executions":     execDelta,
		"last_executed":  lastExecuted,
		"profile_source": "osquery_schedule",
	}

	return profile, nil
}

func (p *queryProfiler) scheduledProfilesDiagnostics(ctx context.Context, qe queryExecutor) []byte {
	return p.scheduledProfilesDiagnosticsWithResolver(ctx, qe, nil)
}

func (p *queryProfiler) scheduledProfilesDiagnosticsWithResolver(ctx context.Context, qe queryExecutor, resolveQuery func(name string) (string, bool)) []byte {
	if qe == nil {
		if p.log != nil {
			p.log.Warnw("Failed to collect scheduled query profiles for Agent diagnostics.", "error", "osquery client is not connected")
		}
		return diagnosticsErrorJSON("osquery client is not connected")
	}

	rows, err := qe.Query(ctx, osqueryScheduleProfilesDiagnosticsQuery, 10*time.Second)
	if err != nil {
		if p.log != nil {
			p.log.Warnw("Failed to collect scheduled query profiles for Agent diagnostics.", "error", err)
		}
		return diagnosticsErrorJSON(fmt.Sprintf("failed to query osquery_schedule: %v", err))
	}

	profiles := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		queryName := toString(row["name"])
		queryText := ""
		if resolveQuery != nil {
			if q, ok := resolveQuery(queryName); ok {
				queryText = q
			}
		}
		profiles = append(profiles, scheduledProfileFromScheduleRow(queryName, queryText, row))
	}

	payload := map[string]interface{}{
		"generated_at": time.Now().UTC().Format(time.RFC3339Nano),
		"profiles":     profiles,
		"count":        len(profiles),
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		if p.log != nil {
			p.log.Warnw("Failed to collect scheduled query profiles for Agent diagnostics.", "error", err)
		}
		return diagnosticsErrorJSON(err.Error())
	}
	return data
}

func diagnosticsErrorJSON(message string) []byte {
	payload := map[string]interface{}{
		"error": message,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return []byte(`{"error":"failed to marshal diagnostics error payload"}`)
	}
	return data
}

func scheduledProfileFromScheduleRow(queryName, queryText string, row map[string]interface{}) map[string]interface{} {
	executions := toInt64(row["executions"])
	if executions <= 0 {
		executions = 1
	}

	wallTotal := toInt64(row["wall_time_ms"])
	userTotal := toInt64(row["user_time"])
	systemTotal := toInt64(row["system_time"])
	outputSizeTotal := toInt64(row["output_size"])

	wallPerExec := toInt64(row["last_wall_time_ms"])
	if wallPerExec <= 0 && wallTotal > 0 {
		wallPerExec = int64(math.Round(float64(wallTotal) / float64(executions)))
	}

	userPerExec := toInt64(row["last_user_time"])
	if userPerExec <= 0 && userTotal > 0 {
		userPerExec = int64(math.Round(float64(userTotal) / float64(executions)))
	}

	systemPerExec := toInt64(row["last_system_time"])
	if systemPerExec <= 0 && systemTotal > 0 {
		systemPerExec = int64(math.Round(float64(systemTotal) / float64(executions)))
	}

	lastMemory := toInt64(row["last_memory"])
	if lastMemory == 0 {
		lastMemory = toInt64(row["average_memory"])
	}

	cpuMS := userPerExec + systemPerExec
	outputPerExec := int64(math.Round(float64(outputSizeTotal) / float64(executions)))

	profile := map[string]interface{}{
		"source":         "scheduled",
		"query_name":     queryName,
		"utilization":    utilizationFromMillis(cpuMS, wallPerExec),
		"duration":       millisToSeconds(wallPerExec),
		"memory":         lastMemory,
		"user_time":      millisToSeconds(userPerExec),
		"system_time":    millisToSeconds(systemPerExec),
		"cpu_time":       millisToSeconds(cpuMS),
		"output_size":    outputPerExec,
		"executions":     toInt64(row["executions"]),
		"last_executed":  toInt64(row["last_executed"]),
		"profile_source": "osquery_schedule",
	}
	if queryText != "" {
		profile["query"] = queryText
	}
	return profile
}

func collectRuntimeSnapshot(ctx context.Context, qe queryExecutor) (runtimeSnapshot, error) {
	var snap runtimeSnapshot
	rows, err := qe.Query(ctx, `
SELECT
  p.pid,
  p.resident_size,
  p.user_time,
  p.system_time,
  (SELECT count(*) FROM process_open_files WHERE pid = p.pid) AS fds
FROM processes p
JOIN osquery_info o ON p.pid = o.pid
LIMIT 1`, 5*time.Second)
	if err != nil {
		return snap, err
	}
	if len(rows) == 0 {
		return snap, fmt.Errorf("osquery process metrics not found")
	}
	row := rows[0]
	snap.pid = toInt64(row["pid"])
	snap.residentSize = toInt64(row["resident_size"])
	snap.userTimeMS = toInt64(row["user_time"])
	snap.systemTimeMS = toInt64(row["system_time"])
	snap.fds = toInt64(row["fds"])
	return snap, nil
}

func buildLiveQueryProfile(query string, before, after runtimeSnapshot, duration time.Duration, hitCount int, queryErr error) map[string]interface{} {
	userDelta := after.userTimeMS - before.userTimeMS
	if userDelta < 0 {
		userDelta = 0
	}
	systemDelta := after.systemTimeMS - before.systemTimeMS
	if systemDelta < 0 {
		systemDelta = 0
	}
	cpuMS := userDelta + systemDelta
	wallMS := duration.Milliseconds()
	if wallMS < 0 {
		wallMS = 0
	}

	exitCode := int64(0)
	if queryErr != nil {
		exitCode = 1
	}

	return map[string]interface{}{
		"source":      "live",
		"query":       query,
		"rows":        hitCount,
		"utilization": utilizationFromMillis(cpuMS, wallMS),
		"duration":    duration.Seconds(),
		"memory":      after.residentSize,
		"user_time":   millisToSeconds(userDelta),
		"system_time": millisToSeconds(systemDelta),
		"cpu_time":    millisToSeconds(cpuMS),
		"fds":         after.fds,
		"exit":        exitCode,
	}
}

func utilizationFromMillis(cpuMS, wallMS int64) float64 {
	if cpuMS <= 0 || wallMS <= 0 {
		return 0
	}
	return float64(cpuMS) / float64(wallMS) * 100.0
}

func millisToSeconds(v int64) float64 {
	return float64(v) / 1000.0
}

func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	case uint:
		return int64(n)
	case uint32:
		return int64(n)
	case uint64:
		return int64(n)
	case float32:
		return int64(n)
	case float64:
		return int64(n)
	case string:
		i, err := strconv.ParseInt(n, 10, 64)
		if err == nil {
			return i
		}
	}
	return 0
}

func toString(v interface{}) string {
	switch s := v.(type) {
	case string:
		return s
	default:
		return fmt.Sprintf("%v", v)
	}
}
