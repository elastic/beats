// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"context"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs/codec/json"
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

			event, err := parseEvent(&log)

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

// The exporter must send what the standalone beat's logstash output sends for
// the same event. The json codec writes `@timestamp` and `@metadata` (with
// `beat`, `version` and `type`) itself; top-level fields with these names are
// part of the event (Kibana's JSON logs carry `type`, for example).
func TestParseEventEncodesLikeLogstashOutput(t *testing.T) {
	ts := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	newFields := func() map[string]any {
		return map[string]any{
			"message": "Server running at http://0.0.0.0:5601",
			"type":    "log",
			"version": "1.2.3",
			"beat":    map[string]any{"name": "proxy", "hostname": "host-1"},
		}
	}

	tests := []struct {
		name string
		// event metadata of the standalone beat; with include_metadata, otelconsumer
		// adds it to the log record body as `@metadata`, together with beat/version/type
		meta            mapstr.M
		includeMetadata bool
	}{
		{name: "without include_metadata"},
		{
			name:            "with include_metadata",
			meta:            mapstr.M{"input_id": "filestream-1", "raw_index": "logs-generic-default"},
			includeMetadata: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := json.New("9.5.0", json.Config{})

			// Standalone beat: the logstash output encodes the beat.Event.
			want, err := enc.Encode("filebeat", &beat.Event{Timestamp: ts, Meta: tt.meta.Clone(), Fields: newFields()})
			require.NoError(t, err)
			wantJSON := string(want) // Encode reuses its buffer

			// Beat receiver: otelconsumer puts the event fields, @timestamp and (with
			// include_metadata) @metadata into the log record body, the exporter turns
			// it back into a beat.Event.
			body := newFields()
			body[beat.TimestampFieldKey] = "2026-10-08T12:00:00.000Z"
			if tt.includeMetadata {
				meta := map[string]any{"beat": "filebeat", "version": "9.5.0", "type": "_doc"}
				maps.Copy(meta, tt.meta)
				body[beat.MetadataFieldKey] = meta
			}
			lr := plog.NewLogRecord()
			require.NoError(t, lr.Body().SetEmptyMap().FromRaw(body))
			event, err := parseEvent(&lr)
			require.NoError(t, err)
			got, err := enc.Encode("filebeat", &event)
			require.NoError(t, err)

			assert.JSONEq(t, wantJSON, string(got))
			// JSONEq doesn't see duplicate keys
			assert.Equal(t, 1, strings.Count(string(got), `"@timestamp"`), "duplicate @timestamp in %s", got)
			assert.Equal(t, 1, strings.Count(string(got), `"@metadata"`), "duplicate @metadata in %s", got)
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
