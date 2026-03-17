// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

const osqueryScheduleProfileQueryPrefix = `
SELECT
  name,
  query,
  executions,
  output_size,
  last_wall_time_ms,
  last_user_time,
  last_system_time,
  last_memory
FROM osquery_schedule
WHERE name = '`

const osqueryScheduleProfileQuerySuffix = `'`

const osqueryScheduleProfilesDiagnosticsQuery = `
SELECT *
FROM osquery_schedule
`

type runtimeSnapshot struct {
	pid          int64
	residentSize int64
	userTimeMS   int64
	systemTimeMS int64
}

type queryProfiler struct {
	log *logp.Logger
}

func newQueryProfiler(log *logp.Logger) *queryProfiler {
	return &queryProfiler{
		log: log,
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
	queryText := toString(row["query"])
	executions := toInt64(row["executions"])
	outputSizeTotal := toInt64(row["output_size"])

	wallMS := toInt64(row["last_wall_time_ms"])
	userMS := toInt64(row["last_user_time"])
	systemMS := toInt64(row["last_system_time"])
	lastMemory := toInt64(row["last_memory"])

	cpuMS := userMS + systemMS
	profile := map[string]interface{}{
		"source":                 "scheduled",
		"query_name":             queryName,
		"utilization":            utilizationFromMillis(cpuMS, wallMS),
		"duration":               wallMS,
		"memory":                 lastMemory,
		"user_time":              userMS,
		"system_time":            systemMS,
		"cpu_time":               cpuMS,
		"output_size_cumulative": outputSizeTotal,
		"executions":             executions,
	}
	if queryText != "" {
		profile["query"] = queryText
	}

	return profile, nil
}

func (p *queryProfiler) scheduledProfilesDiagnostics(ctx context.Context, qe queryExecutor) []byte {
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

	payload := map[string]interface{}{
		"generated_at":     time.Now().UTC().Format(time.RFC3339Nano),
		"osquery_schedule": rows,
		"count":            len(rows),
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

func collectRuntimeSnapshot(ctx context.Context, qe queryExecutor) (runtimeSnapshot, error) {
	var snap runtimeSnapshot
	rows, err := qe.Query(ctx, `
SELECT
  p.pid,
  p.resident_size,
  p.user_time,
  p.system_time
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
	return snap, nil
}

func buildLiveQueryProfile(query string, before, after runtimeSnapshot, duration time.Duration, queryErr error) map[string]interface{} {
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
		"utilization": utilizationFromMillis(cpuMS, wallMS),
		"duration":    wallMS,
		"memory":      after.residentSize,
		"user_time":   userDelta,
		"system_time": systemDelta,
		"cpu_time":    cpuMS,
		"exit":        exitCode,
	}
}

func utilizationFromMillis(cpuMS, wallMS int64) float64 {
	if cpuMS <= 0 || wallMS <= 0 {
		return 0
	}
	return float64(cpuMS) / float64(wallMS) * 100.0
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
