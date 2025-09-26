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
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/otelbeat/otelctx"
	"github.com/elastic/beats/v7/libbeat/publisher"
)

func TestNewLogBatch(t *testing.T) {
	tests := []struct {
		name      string
		setupCtx  func() context.Context
		setupLogs func() plog.Logs
		wantErr   bool
	}{
		{
			name: "valid logs batch",
			setupCtx: func() context.Context {
				ctx := t.Context()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						otelctx.BeatNameCtxKey:        {"filebeat"},
						otelctx.BeatVersionCtxKey:     {"8.0.0"},
						otelctx.BeatIndexPrefixCtxKey: {"filebeat"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			setupLogs: func() plog.Logs {
				logs := plog.NewLogs()
				rl := logs.ResourceLogs().AppendEmpty()
				sl := rl.ScopeLogs().AppendEmpty()

				// First log record
				lr1 := sl.LogRecords().AppendEmpty()
				lr1.SetObservedTimestamp(pcommon.NewTimestampFromTime(mustParseTime("2023-01-01T12:00:00Z")))
				bodyMap1 := lr1.Body().SetEmptyMap()
				bodyMap1.PutStr("message", "test message 1")
				bodyMap1.PutStr(beat.TimestampFieldKey, "2023-01-01T12:00:00.000Z")

				// Second log record
				lr2 := sl.LogRecords().AppendEmpty()
				lr2.SetObservedTimestamp(pcommon.NewTimestampFromTime(mustParseTime("2023-01-01T12:01:00Z")))
				bodyMap2 := lr2.Body().SetEmptyMap()
				bodyMap2.PutStr("message", "test message 2")
				bodyMap2.PutStr(beat.TimestampFieldKey, "2023-01-01T12:01:00.000Z")

				return logs
			},
			wantErr: false,
		},
		{
			name: "empty logs batch",
			setupCtx: func() context.Context {
				ctx := t.Context()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						otelctx.BeatNameCtxKey:        {"filebeat"},
						otelctx.BeatVersionCtxKey:     {"8.0.0"},
						otelctx.BeatIndexPrefixCtxKey: {"filebeat"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			setupLogs: func() plog.Logs {
				return plog.NewLogs()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			logs := tt.setupLogs()

			batch, err := NewLogBatch(ctx, logs)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, batch)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, batch)
				assert.NotNil(t, batch.resultCh)
				assert.Equal(t, uint64(0), batch.NumRetries())

				// Verify events count matches input
				expectedEventCount := logs.LogRecordCount()
				assert.Len(t, batch.pendingEvents, expectedEventCount)
			}
		})
	}
}

func TestCreateEvents(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		setup    func() plog.Logs
		wantErr  bool
	}{
		{
			name: "multiple resource logs with multiple scope logs",
			setupCtx: func() context.Context {
				ctx := t.Context()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						otelctx.BeatNameCtxKey:        {"filebeat"},
						otelctx.BeatVersionCtxKey:     {"9.0.0"},
						otelctx.BeatIndexPrefixCtxKey: {"filebeat"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			setup: func() plog.Logs {
				logs := plog.NewLogs()

				// First resource log
				rl1 := logs.ResourceLogs().AppendEmpty()
				sl1 := rl1.ScopeLogs().AppendEmpty()
				lr1 := sl1.LogRecords().AppendEmpty()
				lr1.SetObservedTimestamp(pcommon.NewTimestampFromTime(mustParseTime("2023-01-01T12:00:00Z")))
				bodyMap1 := lr1.Body().SetEmptyMap()
				msg1 := "resource1 scope1 log1"
				bodyMap1.PutStr("message", msg1)

				// Second resource log
				rl2 := logs.ResourceLogs().AppendEmpty()
				sl2 := rl2.ScopeLogs().AppendEmpty()
				lr2 := sl2.LogRecords().AppendEmpty()
				lr2.SetObservedTimestamp(pcommon.NewTimestampFromTime(mustParseTime("2023-01-01T12:01:00Z")))
				bodyMap2 := lr2.Body().SetEmptyMap()
				msg2 := "resource2 scope1 log1"
				bodyMap2.PutStr("message", msg2)

				// Second scope in second resource log
				sl3 := rl2.ScopeLogs().AppendEmpty()
				lr3 := sl3.LogRecords().AppendEmpty()
				lr3.SetObservedTimestamp(pcommon.NewTimestampFromTime(mustParseTime("2023-01-01T12:02:00Z")))
				bodyMap3 := lr3.Body().SetEmptyMap()
				msg3 := "resource2 scope2 log1"
				bodyMap3.PutStr("message", msg3)

				return logs
			},
			wantErr: false,
		},
		{
			name: "empty logs",
			setupCtx: func() context.Context {
				ctx := t.Context()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						"beat_name":    {"metricbeat"},
						"beat_version": {"7.15.2"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			setup: func() plog.Logs {
				return plog.NewLogs()
			},
			wantErr: false,
		},
		{
			name: "invalid beats event in logs",
			setupCtx: func() context.Context {
				ctx := t.Context()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						"beat_version": {"8.0.0"}, // Missing beat_name
					}),
				}
				return client.NewContext(ctx, info)
			},
			setup: func() plog.Logs {
				logs := plog.NewLogs()
				rl := logs.ResourceLogs().AppendEmpty()
				sl := rl.ScopeLogs().AppendEmpty()

				lr := sl.LogRecords().AppendEmpty()
				lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(mustParseTime("2023-01-01T12:00:00Z")))
				bodyMap := lr.Body().SetEmptyMap()
				bodyMap.PutStr("message", "test message")

				return logs
			},
			wantErr: true,
		},
		{
			name: "invalid log record body",
			setupCtx: func() context.Context {
				ctx := t.Context()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						"beat_name":    {"filebeat"},
						"beat_version": {"8.0.0"},
					}),
				}
				return client.NewContext(ctx, info)
			},
			setup: func() plog.Logs {
				logs := plog.NewLogs()
				rl := logs.ResourceLogs().AppendEmpty()
				sl := rl.ScopeLogs().AppendEmpty()

				lr := sl.LogRecords().AppendEmpty()
				lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(mustParseTime("2023-01-01T12:00:00Z")))
				lr.Body().SetStr("not a map") // Invalid body type

				return logs
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			logs := tt.setup()

			events, err := createEvents(ctx, &logs)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, events)
			} else {
				require.NoError(t, err)
				assert.Len(t, events, logs.LogRecordCount())

				ctxData := client.FromContext(ctx)
				eventIndex := 0

				// Verify each event matches the original plog data and metadata
				for _, rl := range logs.ResourceLogs().All() {
					for _, sl := range rl.ScopeLogs().All() {
						for _, lr := range sl.LogRecords().All() {
							beatEvent := events[eventIndex].Content

							// Verify metadata from context
							assert.Equal(t, ctxData.Metadata.Get("beat_name")[0], beatEvent.Meta["beat"])
							assert.Equal(t, ctxData.Metadata.Get("beat_version")[0], beatEvent.Meta["version"])

							// Verify fields match original log record body
							originalBody := lr.Body().Map().AsRaw()
							for key, expectedValue := range originalBody {
								assert.Equal(t, expectedValue, beatEvent.Fields[key],
									"Field %s should match original log record", key)
							}

							// Verify timestamp is set
							assert.Equal(t, lr.ObservedTimestamp().AsTime(), beatEvent.Timestamp)

							eventIndex++
						}
					}
				}
			}
		})
	}
}

func TestLogBatchEvents(t *testing.T) {
	batch, err := NewLogBatch(t.Context(), plog.NewLogs())
	require.NoError(t, err)

	batch.pendingEvents = []publisher.Event{
		{Content: beat.Event{Fields: map[string]any{"message": "test1"}}},
		{Content: beat.Event{Fields: map[string]any{"message": "test2"}}},
	}

	events := batch.Events()

	assert.Len(t, events, 2)
	assert.Equal(t, batch.pendingEvents, events)
}

func TestLogBatchACK(t *testing.T) {
	batch, err := NewLogBatch(t.Context(), plog.NewLogs())
	require.NoError(t, err)

	batch.ACK()

	var result LogBatchResult
	select {
	case result = <-batch.Result():
	default:
		t.Fatal("no ACK result received")
	}

	assert.Equal(t, LogBatchResultACK, result)
	assert.Equal(t, uint64(0), batch.NumRetries())
}

func TestLogBatchDrop(t *testing.T) {
	batch, err := NewLogBatch(t.Context(), plog.NewLogs())
	require.NoError(t, err)

	batch.Drop()

	var result LogBatchResult
	select {
	case result = <-batch.Result():
	default:
		t.Fatal("no Drop result received")
	}

	assert.Equal(t, LogBatchResultDrop, result)
	assert.Equal(t, uint64(0), batch.NumRetries())
}

func TestLogBatchRetry(t *testing.T) {
	var result LogBatchResult
	batch, err := NewLogBatch(t.Context(), plog.NewLogs())
	require.NoError(t, err)

	// First retry
	batch.Retry()

	select {
	case result = <-batch.Result():
	default:
		t.Fatal("first Retry result not received")
	}

	assert.Equal(t, LogBatchResultRetry, result)
	assert.Equal(t, uint64(1), batch.NumRetries())

	// Second retry
	batch.Retry()

	select {
	case result = <-batch.Result():
	default:
		t.Fatal("second Retry result not received")
	}

	assert.Equal(t, LogBatchResultRetry, result)
	assert.Equal(t, uint64(2), batch.NumRetries())
}

func TestLogBatchRetryEvents(t *testing.T) {
	events := []publisher.Event{
		{Content: beat.Event{Fields: map[string]any{"message": "test1"}}},
		{Content: beat.Event{Fields: map[string]any{"message": "test2"}}},
	}

	batch, err := NewLogBatch(t.Context(), plog.NewLogs())
	require.NoError(t, err)

	batch.pendingEvents = events

	retryEvents := []publisher.Event{events[1]}
	batch.RetryEvents(retryEvents)

	var result LogBatchResult
	select {
	case result = <-batch.Result():
	default:
		t.Fatal("no RetryEvents result received")
	}

	assert.Equal(t, LogBatchResultRetry, result)
	assert.Equal(t, retryEvents, batch.Events())
	assert.Equal(t, uint64(1), batch.NumRetries())
}

func TestLogBatchCancelled(t *testing.T) {
	batch, err := NewLogBatch(t.Context(), plog.NewLogs())
	require.NoError(t, err)

	batch.Cancelled()

	var result LogBatchResult
	select {
	case result = <-batch.Result():
	default:
		t.Fatal("no Cancelled result received")
	}

	assert.Equal(t, LogBatchResultCancelled, result)
	assert.Equal(t, uint64(0), batch.NumRetries())
}

func TestSplitRetry(t *testing.T) {
	batch, err := NewLogBatch(t.Context(), plog.NewLogs())
	require.NoError(t, err)
	assert.False(t, batch.SplitRetry())
}

func TestAddRetry(t *testing.T) {
	tests := []struct {
		name     string
		current  uint64
		delta    int
		expected uint64
	}{
		{"add zero", 5, 0, 5},
		{"add to zero", 3, 0, 3},
		{"add to non-zero", 5, 5, 10},
		{"sub from zero", 0, -1, 0},
		{"sub from non-zero", 2, -1, 1},
		{"sub from smaller value", 2, -5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch, err := NewLogBatch(t.Context(), plog.NewLogs())
			require.NoError(t, err)

			batch.retries.Store(tt.current)
			batch.AddRetry(tt.delta)
			assert.Equal(t, tt.expected, batch.NumRetries())
		})
	}
}

func mustParseTime(timeStr string) time.Time {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		panic(err)
	}
	return t
}
