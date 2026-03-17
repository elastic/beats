// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestToInt64(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want int64
	}{
		{"int", 42, 42},
		{"int64", int64(100), 100},
		{"float64", float64(3.14), 3},
		{"string", "999", 999},
		{"string invalid", "abc", 0},
		{"nil", nil, 0},
		{"uint", uint(10), 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toInt64(tt.in)
			if got != tt.want {
				t.Errorf("toInt64(%v) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want string
	}{
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"nil", nil, "<nil>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toString(tt.in)
			if got != tt.want {
				t.Errorf("toString(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestUtilizationFromMillis(t *testing.T) {
	tests := []struct {
		cpuMS, wallMS int64
		want          float64
	}{
		{0, 100, 0},
		{100, 0, 0},
		{50, 100, 50},
		{100, 100, 100},
		{25, 100, 25},
	}
	for _, tt := range tests {
		got := utilizationFromMillis(tt.cpuMS, tt.wallMS)
		if got != tt.want {
			t.Errorf("utilizationFromMillis(%d, %d) = %f, want %f", tt.cpuMS, tt.wallMS, got, tt.want)
		}
	}
}

func TestMillisToSeconds(t *testing.T) {
	if got := millisToSeconds(1000); got != 1.0 {
		t.Errorf("millisToSeconds(1000) = %f, want 1.0", got)
	}
	if got := millisToSeconds(0); got != 0 {
		t.Errorf("millisToSeconds(0) = %f, want 0", got)
	}
}

func TestBuildLiveQueryProfile(t *testing.T) {
	before := runtimeSnapshot{pid: 1, residentSize: 1000, userTimeMS: 10, systemTimeMS: 5, fds: 10}
	after := runtimeSnapshot{pid: 1, residentSize: 2000, userTimeMS: 20, systemTimeMS: 15, fds: 12}
	duration := 100 * time.Millisecond

	profile := buildLiveQueryProfile("SELECT 1", before, after, duration, 5, nil)
	if profile["source"] != "live" {
		t.Errorf("source = %v, want live", profile["source"])
	}
	if profile["query"] != "SELECT 1" {
		t.Errorf("query = %v, want SELECT 1", profile["query"])
	}
	if profile["rows"] != 5 {
		t.Errorf("rows = %v, want 5", profile["rows"])
	}
	if profile["exit"].(int64) != 0 {
		t.Errorf("exit = %v, want 0", profile["exit"])
	}
	if profile["memory"] != int64(2000) {
		t.Errorf("memory = %v, want 2000", profile["memory"])
	}
	if profile["fds"] != int64(12) {
		t.Errorf("fds = %v, want 12", profile["fds"])
	}

	profileErr := buildLiveQueryProfile("SELECT 1", before, after, duration, 0, context.DeadlineExceeded)
	if profileErr["exit"].(int64) != 1 {
		t.Errorf("exit on error = %v, want 1", profileErr["exit"])
	}
}

func TestScheduledProfileFromScheduleRow(t *testing.T) {
	row := map[string]interface{}{
		"name":              "test_query",
		"executions":        int64(10),
		"last_executed":     int64(12345),
		"wall_time_ms":      int64(5000),
		"last_wall_time_ms": int64(600),
		"user_time":         int64(2000),
		"last_user_time":    int64(200),
		"system_time":       int64(1000),
		"last_system_time":  int64(100),
		"output_size":       int64(10000),
		"last_memory":       int64(50000),
	}
	profile := scheduledProfileFromScheduleRow("test_query", "SELECT * FROM foo", row)
	if profile["query_name"] != "test_query" {
		t.Errorf("query_name = %v, want test_query", profile["query_name"])
	}
	if profile["query"] != "SELECT * FROM foo" {
		t.Errorf("query = %v, want SELECT * FROM foo", profile["query"])
	}
	if profile["source"] != "scheduled" {
		t.Errorf("source = %v, want scheduled", profile["source"])
	}
	if profile["profile_source"] != "osquery_schedule" {
		t.Errorf("profile_source = %v, want osquery_schedule", profile["profile_source"])
	}
	if profile["executions"] != int64(10) {
		t.Errorf("executions = %v, want 10", profile["executions"])
	}
}

func TestDiagnosticsErrorJSON(t *testing.T) {
	data := diagnosticsErrorJSON("something failed")
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m["error"] != "something failed" {
		t.Errorf("error = %v, want something failed", m["error"])
	}
}

func TestNewQueryProfiler(t *testing.T) {
	p := newQueryProfiler(logp.NewLogger("test"))
	if p == nil {
		t.Fatal("newQueryProfiler() returned nil")
	}
}

type mockProfileQueryExecutor struct {
	rows []map[string]interface{}
	err  error
}

func (m *mockProfileQueryExecutor) Query(ctx context.Context, sql string, timeout time.Duration) ([]map[string]interface{}, error) {
	return m.rows, m.err
}

func TestProfileScheduledQuery_FirstRun(t *testing.T) {
	ctx := context.Background()
	qe := &mockProfileQueryExecutor{
		rows: []map[string]interface{}{
			{
				"name":              "q1",
				"executions":        int64(1),
				"last_executed":     int64(999),
				"output_size":       int64(100),
				"wall_time_ms":      int64(50),
				"last_wall_time_ms": int64(50),
				"user_time":         int64(20),
				"last_user_time":    int64(20),
				"system_time":       int64(10),
				"last_system_time":  int64(10),
				"last_memory":       int64(1000),
			},
		},
	}
	p := newQueryProfiler(logp.NewLogger("test"))
	profile, err := p.profileScheduledQuery(ctx, qe, "q1")
	if err != nil {
		t.Fatal(err)
	}
	if profile["query_name"] != "q1" {
		t.Errorf("query_name = %v, want q1", profile["query_name"])
	}
	if profile["executions"] != int64(1) {
		t.Errorf("executions = %v, want 1", profile["executions"])
	}
	if profile["duration"] != 0.05 {
		t.Errorf("duration = %v, want 0.05", profile["duration"])
	}
}

func TestProfileScheduledQuery_ExecutionReset(t *testing.T) {
	ctx := context.Background()
	// Simulate osquery restart: first call with high counts, second with low counts.
	p := newQueryProfiler(logp.NewLogger("test"))
	qeFirst := &mockProfileQueryExecutor{
		rows: []map[string]interface{}{
			{
				"name":              "q1",
				"executions":        int64(100),
				"last_executed":     int64(999),
				"output_size":       int64(10000),
				"wall_time_ms":      int64(5000),
				"last_wall_time_ms": int64(60),
				"user_time":         int64(2000),
				"last_user_time":    int64(20),
				"system_time":       int64(1000),
				"last_system_time":  int64(10),
				"last_memory":       int64(5000),
			},
		},
	}
	_, err := p.profileScheduledQuery(ctx, qeFirst, "q1")
	if err != nil {
		t.Fatal(err)
	}
	// Second run: executions dropped (e.g. osqueryd restarted), should not produce negative per-exec values.
	qeSecond := &mockProfileQueryExecutor{
		rows: []map[string]interface{}{
			{
				"name":              "q1",
				"executions":        int64(1),
				"last_executed":     int64(1000),
				"output_size":       int64(100),
				"wall_time_ms":      int64(50),
				"last_wall_time_ms": int64(50),
				"user_time":         int64(20),
				"last_user_time":    int64(20),
				"system_time":       int64(10),
				"last_system_time":  int64(10),
				"last_memory":       int64(1000),
			},
		},
	}
	profile, err := p.profileScheduledQuery(ctx, qeSecond, "q1")
	if err != nil {
		t.Fatal(err)
	}
	// After reset we treat as single run; duration should be 0.05 (50ms), not negative/zero.
	if d, ok := profile["duration"].(float64); !ok || d < 0 {
		t.Errorf("duration = %v, expected non-negative float", profile["duration"])
	}
	if profile["executions"] != int64(1) {
		t.Errorf("executions = %v, want 1", profile["executions"])
	}
}

func TestScheduledProfilesDiagnosticsWithResolver_NilExecutor(t *testing.T) {
	ctx := context.Background()
	p := newQueryProfiler(logp.NewLogger("test"))
	data := p.scheduledProfilesDiagnosticsWithResolver(ctx, nil, nil)
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m["error"] == nil {
		t.Error("expected error key when executor is nil")
	}
}
