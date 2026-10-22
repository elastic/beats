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
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
			name: "event with top-level beat, version and type fields",
			setupLog: func() plog.LogRecord {
				lr := plog.NewLogRecord()
				lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))

				bodyMap := lr.Body().SetEmptyMap()
				bodyMap.PutStr("message", "test message")
				bodyMap.PutStr("type", "log")
				bodyMap.PutStr("version", "1.2.3")
				bodyMap.PutEmptyMap("beat").PutStr("name", "proxy")

				return lr
			},
			wantErr: false,
		},
		{
			name: "invalid event body - not a map",
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
			log := tt.setupLog()

			event, err := parseEvent(&log, "")

			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, consumererror.IsPermanent(err))
			} else {

				require.NoError(t, err)

				// Verify fields match original log record body. `@timestamp` moves to
				// event.Timestamp (the json codec writes it from there).
				originalBody := log.Body().Map().AsRaw()
				for key, expectedValue := range originalBody {
					if key == beat.TimestampFieldKey {
						assert.NotContains(t, event.Fields, key)
						continue
					}
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

func TestExtractMetadataFields(t *testing.T) {
	tests := map[string]struct {
		fields       map[string]any
		index        string
		expectedMeta mapstr.M
	}{
		"no @metadata": {
			fields:       map[string]any{"message": "test"},
			expectedMeta: nil,
		},
		"version and type are removed, beat is kept": {
			fields: map[string]any{
				beat.MetadataFieldKey: map[string]any{"beat": "filebeat", "version": "9.6.0", "type": "_doc", "pipeline": "p"},
			},
			expectedMeta: mapstr.M{"beat": "filebeat", "pipeline": "p"},
		},
		"explicit index overrides beat": {
			fields: map[string]any{
				beat.MetadataFieldKey: map[string]any{"beat": "filebeat"},
			},
			index:        "My-Index",
			expectedMeta: mapstr.M{"beat": "my-index"},
		},
		"explicit index is set when beat is missing": {
			fields: map[string]any{
				beat.MetadataFieldKey: map[string]any{"pipeline": "p"},
			},
			index:        "my-index",
			expectedMeta: mapstr.M{"beat": "my-index", "pipeline": "p"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			meta := extractMetadataFields(tt.fields, tt.index)
			assert.Equal(t, tt.expectedMeta, meta, "unexpected event metadata")
			assert.NotContains(t, tt.fields, beat.MetadataFieldKey, "@metadata should be removed from the event fields")
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
