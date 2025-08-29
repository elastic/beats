// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/elastic/beats/v7/libbeat/beat"
)

func TestParseEvent(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		setupLog func() plog.LogRecord
		wantErr  bool
	}{
		{
			name: "valid beats event with timestamp",
			setupCtx: func() context.Context {
				ctx := context.Background()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						"beat_name":    {"filebeat"},
						"beat_version": {"8.0.0"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			setupLog: func() plog.LogRecord {
				lr := plog.NewLogRecord()
				lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))

				bodyMap := lr.Body().SetEmptyMap()
				bodyMap.PutStr("message", "test message")
				bodyMap.PutStr(beat.TimestampFieldKey, "2023-01-01T12:00:00.000Z")

				return lr
			},
			wantErr: false,
		},
		{
			name: "valid beats event without timestamp",
			setupCtx: func() context.Context {
				ctx := context.Background()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						"beat_name":    {"filebeat"},
						"beat_version": {"8.0.0"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			setupLog: func() plog.LogRecord {
				lr := plog.NewLogRecord()
				observedTime := time.Now()
				lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(observedTime))

				bodyMap := lr.Body().SetEmptyMap()
				bodyMap.PutStr("message", "test message")

				return lr
			},
			wantErr: false,
		},
		{
			name: "invalid beats event metadata - missing beat name",
			setupCtx: func() context.Context {
				ctx := context.Background()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						"beat_version": {"8.0.0"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			setupLog: func() plog.LogRecord {
				lr := plog.NewLogRecord()
				lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))

				bodyMap := lr.Body().SetEmptyMap()
				bodyMap.PutStr("message", "test message")

				return lr
			},
			wantErr: true,
		},
		{
			name: "invalid beats event metadata - empty beat name",
			setupCtx: func() context.Context {
				ctx := context.Background()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						"beat_name":    {""},
						"beat_version": {"8.0.0"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			setupLog: func() plog.LogRecord {
				lr := plog.NewLogRecord()
				lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))

				bodyMap := lr.Body().SetEmptyMap()
				bodyMap.PutStr("message", "test message")

				return lr
			},
			wantErr: true,
		},
		{
			name: "invalid event body - not a map",
			setupCtx: func() context.Context {
				ctx := context.Background()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						"beat_name":    {"filebeat"},
						"beat_version": {"8.0.0"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			setupLog: func() plog.LogRecord {
				lr := plog.NewLogRecord()
				lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))
				lr.Body().SetStr("not a map")

				return lr
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			log := tt.setupLog()

			event, err := parseEvent(ctx, &log)

			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, consumererror.IsPermanent(err))
			} else {
				ctxData := client.FromContext(ctx)

				require.NoError(t, err)

				// Verify metadata from context
				assert.Equal(t, ctxData.Metadata.Get("beat_name")[0], event.Meta["beat"])
				assert.Equal(t, ctxData.Metadata.Get("beat_version")[0], event.Meta["version"])

				// Verify fields match original log record body
				originalBody := log.Body().Map().AsRaw()
				for key, expectedValue := range originalBody {
					assert.Equal(t, expectedValue, event.Fields[key],
						"Field %s should match original log record", key)
				}

				// Verify timestamp against ObservedTimestamp if body has no `@timestamp`
				if _, ok := originalBody[beat.TimestampFieldKey]; !ok {
					assert.Equal(t, log.ObservedTimestamp().AsTime(), event.Timestamp)
				}
			}
		})
	}
}

func TestParseEventFields(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() plog.LogRecord
		wantOk   bool
		expected map[string]any
	}{
		{
			name: "valid map body",
			setup: func() plog.LogRecord {
				lr := plog.NewLogRecord()
				bodyMap := lr.Body().SetEmptyMap()
				bodyMap.PutStr("message", "test message")
				bodyMap.PutInt("count", 42)
				return lr
			},
			wantOk: true,
			expected: map[string]any{
				"message": "test message",
				"count":   int64(42),
			},
		},
		{
			name: "non-map body - string",
			setup: func() plog.LogRecord {
				lr := plog.NewLogRecord()
				lr.Body().SetStr("not a map")
				return lr
			},
			wantOk:   false,
			expected: nil,
		},
		{
			name: "non-map body - int",
			setup: func() plog.LogRecord {
				lr := plog.NewLogRecord()
				lr.Body().SetInt(123)
				return lr
			},
			wantOk:   false,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := tt.setup()

			fields, ok := parseEventFields(&record)

			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.expected, fields)
		})
	}
}

func TestParseEventTimestamp(t *testing.T) {
	tests := []struct {
		name         string
		body         map[string]any
		wantOk       bool
		expectedTime time.Time
	}{
		{
			name: "valid timestamp string",
			body: map[string]any{
				beat.TimestampFieldKey: "2023-01-01T12:00:00.000Z",
			},
			wantOk:       true,
			expectedTime: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			name: "missing timestamp field",
			body: map[string]any{
				"message": "test",
			},
			wantOk:       false,
			expectedTime: time.Time{},
		},
		{
			name: "invalid timestamp format",
			body: map[string]any{
				beat.TimestampFieldKey: "2023-01-01 12:00:00",
			},
			wantOk:       false,
			expectedTime: time.Time{},
		},
		{
			name: "timestamp not a string",
			body: map[string]any{
				beat.TimestampFieldKey: 1672574400000,
			},
			wantOk:       false,
			expectedTime: time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timestamp, ok := parseEventTimestamp(tt.body)

			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.expectedTime, timestamp)
		})
	}
}

func TestParseEventMetadata(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		expected map[string]any
	}{
		{
			name: "complete metadata",
			setupCtx: func() context.Context {
				ctx := context.Background()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						"beat_name":    {"filebeat"},
						"beat_version": {"8.0.0"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			expected: map[string]any{
				"beat":    "filebeat",
				"version": "8.0.0",
			},
		},
		{
			name: "missing beat name",
			setupCtx: func() context.Context {
				ctx := context.Background()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						"beat_version": {"8.0.0"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			expected: map[string]any{
				"beat":    "",
				"version": "8.0.0",
			},
		},
		{
			name: "missing beat version",
			setupCtx: func() context.Context {
				ctx := context.Background()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						"beat_name": {"filebeat"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			expected: map[string]any{
				"beat":    "filebeat",
				"version": "",
			},
		},
		{
			name: "no metadata",
			setupCtx: func() context.Context {
				ctx := context.Background()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{}),
				}
				return client.NewContext(ctx, info)
			},
			expected: map[string]any{
				"beat":    "",
				"version": "",
			},
		},
		{
			name: "no client info in context",
			setupCtx: func() context.Context {
				return context.Background()
			},
			expected: map[string]any{
				"beat":    "",
				"version": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()

			metadata := parseEventMetadata(ctx)

			assert.Equal(t, tt.expected, metadata)
		})
	}
}

func TestIsBeatsEvent(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]any
		expected bool
	}{
		{
			name: "valid beats event",
			metadata: map[string]any{
				"beat":    "filebeat",
				"version": "8.0.0",
			},
			expected: true,
		},
		{
			name: "missing beat field",
			metadata: map[string]any{
				"version": "8.0.0",
			},
			expected: false,
		},
		{
			name: "nil beat field",
			metadata: map[string]any{
				"beat":    nil,
				"version": "8.0.0",
			},
			expected: false,
		},
		{
			name: "empty beat field",
			metadata: map[string]any{
				"beat":    "",
				"version": "8.0.0",
			},
			expected: false,
		},
		{
			name:     "empty metadata",
			metadata: map[string]any{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBeatsEvent(tt.metadata)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetBeatVersion(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		expected string
	}{
		{
			name: "version exists",
			setupCtx: func() context.Context {
				ctx := context.Background()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						"beat_name":    {"filebeat"},
						"beat_version": {"8.0.0"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			expected: "8.0.0",
		},
		{
			name: "version missing",
			setupCtx: func() context.Context {
				ctx := context.Background()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						"beat_name": {"filebeat"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			expected: "",
		},
		{
			name: "no client info",
			setupCtx: func() context.Context {
				return context.Background()
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()

			version := GetBeatVersion(ctx)

			assert.Equal(t, tt.expected, version)
		})
	}
}
