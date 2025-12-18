// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beatprocessor

import (
	"context"
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
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
			beatProcessor.ConsumeLogs(context.Background(), logs)
		}
	})
}

type testCase struct {
	name      string
	logRecord func() plog.LogRecord
}
