// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatprocessor

import (
	"context"
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/add_host_metadata"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// Runs the OTel Beat processor with a no-op processor inside
// to measure the overhead of the OTel processor itself.
func BenchmarkBeatProcessor(b *testing.B) {
	testCases := []testCase{
		{
			name: "empty log record",
			logRecord: func() plog.LogRecord {
				logRecord := plog.NewLogRecord()
				logRecord.Body().SetEmptyMap()
				return logRecord
			},
		},
		{
			name: "message",
			logRecord: func() plog.LogRecord {
				logRecord := plog.NewLogRecord()
				logRecord.Body().SetEmptyMap()
				logRecord.Body().Map().PutStr("message", "test log message")
				return logRecord
			},
		},
		{
			name: "Kubernetes log",
			logRecord: func() plog.LogRecord {
				logRecord := plog.NewLogRecord()
				logRecord.Body().SetEmptyMap()
				logRecord.Body().Map().PutStr("time", "2026-01-13T13:01:05.551185505Z")
				logRecord.Body().Map().PutStr("stream", "stderr")
				logRecord.Body().Map().PutStr("log", "W0113 13:01:05.550863       1 warnings.go:70] v1 Endpoints is deprecated in v1.33+; use discovery.k8s.io/v1 EndpointSlice")
				return logRecord
			},
		},
		{
			name: "Nginx access log",
			logRecord: func() plog.LogRecord {
				logRecord := plog.NewLogRecord()
				logRecord.Body().SetEmptyMap()
				logRecord.Body().Map().PutStr("timestamp", "2025-10-05T13:33:14+00:00")
				logRecord.Body().Map().PutStr("pid", "30")
				logRecord.Body().Map().PutStr("client_ip", "172.19.0.1")
				logRecord.Body().Map().PutStr("request_id", "72e5a0eeb44b6608d161254a5eaf1662")
				logRecord.Body().Map().PutStr("http_method", "GET")
				logRecord.Body().Map().PutStr("http_path", "/")
				logRecord.Body().Map().PutStr("protocol", "HTTP/1.1")
				logRecord.Body().Map().PutStr("host", "localhost")
				logRecord.Body().Map().PutStr("user_agent", "curl/8.5.0")
				logRecord.Body().Map().PutStr("referer", "")
				logRecord.Body().Map().PutInt("status_code", 200)
				logRecord.Body().Map().PutInt("bytes_sent", 615)
				logRecord.Body().Map().PutDouble("request_time_secs", 0.01)
				return logRecord
			},
		},
	}

	benchmarkBeatProcessor(b, 0, testCases[0])

	for _, tc := range testCases {
		for _, logCount := range []int{1, 1000} {
			benchmarkBeatProcessor(b, logCount, tc)
		}
	}
}

func benchmarkBeatProcessor(b *testing.B, logCount int, tc testCase) {
	b.Run(fmt.Sprintf("%s/%d_logs", tc.name, logCount), func(b *testing.B) {
		// Prepare logs.
		logs := plog.NewLogs()
		resourceLogs := logs.ResourceLogs().AppendEmpty()
		scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
		for range logCount {
			logRecord := scopeLogs.LogRecords().AppendEmpty()
			tc.logRecord().CopyTo(logRecord)
		}

		// Create Beat processor with a no-op processor inside.
		beatProcessor := &beatProcessor{
			logger: zap.NewNop(),
			processors: []beat.Processor{
				mockProcessor{
					runFunc: func(event *beat.Event) (*beat.Event, error) {
						return event, nil
					},
				},
			},
		}

		for b.Loop() {
			_, _ = beatProcessor.ConsumeLogs(context.Background(), logs)
		}
	})
}

type testCase struct {
	name      string
	logRecord func() plog.LogRecord
}

// legacyOnlyProcessor wraps a beat.Processor and intentionally does NOT implement
// PdataProcessor, forcing the legacy mapstr.M conversion path in benchmarks.
type legacyOnlyProcessor struct {
	beat.Processor
}

// BenchmarkWhenCondition measures the overhead of the conditional (when:) wrapper
// for three scenarios:
//   - condition false: event is skipped entirely (no processor runs)
//   - condition true, pdata inner: condition passes and inner RunPdata is called
//   - condition true, legacy inner: condition passes but inner only supports Run (round-trip)
func BenchmarkWhenCondition(b *testing.B) {
	logger := logp.NewNopLogger()

	hostProc, err := add_host_metadata.New(config.NewConfig(), logger)
	if err != nil {
		b.Fatalf("failed to create add_host_metadata: %v", err)
	}

	// Wrap add_host_metadata with a when.contains.tags condition.
	wrappedPdata, err := processors.NewConditional(func(cfg *config.C, log *logp.Logger) (beat.Processor, error) {
		return hostProc, nil
	})(mustConfig(b, map[string]any{
		"when": map[string]any{"contains": map[string]any{"tags": "forwarded"}},
	}), logger)
	if err != nil {
		b.Fatalf("failed to wrap processor: %v", err)
	}

	// Same condition but inner is legacy-only.
	wrappedLegacy, err := processors.NewConditional(func(cfg *config.C, log *logp.Logger) (beat.Processor, error) {
		return legacyOnlyProcessor{hostProc}, nil
	})(mustConfig(b, map[string]any{
		"when": map[string]any{"contains": map[string]any{"tags": "forwarded"}},
	}), logger)
	if err != nil {
		b.Fatalf("failed to wrap legacy processor: %v", err)
	}

	makeLogs := func(withTag bool) plog.Logs {
		logs := plog.NewLogs()
		lr := logs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
		lr.Body().SetEmptyMap()
		lr.Body().Map().PutStr("message", "test log message")
		if withTag {
			s := lr.Body().Map().PutEmptySlice("tags")
			s.AppendEmpty().SetStr("forwarded")
		}
		return logs
	}

	logsMatch := makeLogs(true)
	logsNoMatch := makeLogs(false)

	b.Run("condition_false", func(b *testing.B) {
		bp := &beatProcessor{logger: zap.NewNop(), processors: []beat.Processor{wrappedPdata}}
		for b.Loop() {
			_, _ = bp.ConsumeLogs(context.Background(), logsNoMatch)
		}
	})

	b.Run("condition_true/pdata_inner", func(b *testing.B) {
		bp := &beatProcessor{logger: zap.NewNop(), processors: []beat.Processor{wrappedPdata}}
		for b.Loop() {
			_, _ = bp.ConsumeLogs(context.Background(), logsMatch)
		}
	})

	b.Run("condition_true/legacy_inner", func(b *testing.B) {
		bp := &beatProcessor{logger: zap.NewNop(), processors: []beat.Processor{wrappedLegacy}}
		for b.Loop() {
			_, _ = bp.ConsumeLogs(context.Background(), logsMatch)
		}
	})
}

func mustConfig(b *testing.B, m map[string]any) *config.C {
	b.Helper()
	cfg, err := config.NewConfigFrom(m)
	if err != nil {
		b.Fatalf("mustConfig: %v", err)
	}
	return cfg
}

// BenchmarkAddHostMetadata compares the legacy mapstr path against the pdata path
// for the add_host_metadata processor to quantify the conversion overhead savings.
func BenchmarkAddHostMetadata(b *testing.B) {
	logger := logp.NewNopLogger()

	proc, err := add_host_metadata.New(config.NewConfig(), logger)
	if err != nil {
		b.Fatalf("failed to create add_host_metadata processor: %v", err)
	}

	testCases := []testCase{
		{
			name: "empty log record",
			logRecord: func() plog.LogRecord {
				lr := plog.NewLogRecord()
				lr.Body().SetEmptyMap()
				return lr
			},
		},
		{
			name: "Nginx access log",
			logRecord: func() plog.LogRecord {
				lr := plog.NewLogRecord()
				lr.Body().SetEmptyMap()
				lr.Body().Map().PutStr("timestamp", "2025-10-05T13:33:14+00:00")
				lr.Body().Map().PutStr("client_ip", "172.19.0.1")
				lr.Body().Map().PutStr("http_method", "GET")
				lr.Body().Map().PutStr("http_path", "/")
				lr.Body().Map().PutInt("status_code", 200)
				return lr
			},
		},
	}

	for _, tc := range testCases {
		for _, logCount := range []int{1, 1000} {
			name := fmt.Sprintf("%s/%d_logs", tc.name, logCount)

			logs := plog.NewLogs()
			rl := logs.ResourceLogs().AppendEmpty()
			sl := rl.ScopeLogs().AppendEmpty()
			for range logCount {
				lr := sl.LogRecords().AppendEmpty()
				tc.logRecord().CopyTo(lr)
			}

				// Legacy path: wrap proc in legacyOnlyProcessor to hide PdataProcessor.
			b.Run("legacy/"+name, func(b *testing.B) {
				bp := &beatProcessor{
					logger:     zap.NewNop(),
					processors: []beat.Processor{legacyOnlyProcessor{proc}},
				}
				for b.Loop() {
					_, _ = bp.ConsumeLogs(context.Background(), logs)
				}
			})

			// Pdata path: processor implements PdataProcessor, fast path is taken.
			b.Run("pdata/"+name, func(b *testing.B) {
				if _, ok := proc.(PdataProcessor); !ok {
					b.Skip("processor does not implement PdataProcessor")
				}
				bp := &beatProcessor{
					logger:     zap.NewNop(),
					processors: []beat.Processor{proc},
				}
				for b.Loop() {
					_, _ = bp.ConsumeLogs(context.Background(), logs)
				}
			})
		}
	}
}
