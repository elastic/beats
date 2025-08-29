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
				ctx := context.Background()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						"beat_name":    {"filebeat"},
						"beat_version": {"8.0.0"},
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
				ctx := context.Background()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						"beat_name":    {"filebeat"},
						"beat_version": {"8.0.0"},
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
				assert.NotNil(t, batch.result)

				// Verify result is initialized correctly
				assert.False(t, batch.result.Acked)
				assert.False(t, batch.result.Dropped)
				assert.False(t, batch.result.Retry)
				assert.False(t, batch.result.Split)
				assert.False(t, batch.result.Cancelled)
				assert.Equal(t, 0, batch.result.Retries)

				// Verify events count matches input
				expectedEventCount := countLogRecords(logs)
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
				ctx := context.Background()
				info := client.Info{
					Metadata: client.NewMetadata(map[string][]string{
						"beat_name":    {"filebeat"},
						"beat_version": {"9.0.0"},
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
				ctx := context.Background()
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
				ctx := context.Background()
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
				ctx := context.Background()
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
				assert.Len(t, events, countLogRecords(logs))

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
	batch := &LogBatch{
		pendingEvents: []publisher.Event{
			{Content: beat.Event{Fields: map[string]any{"message": "test1"}}},
			{Content: beat.Event{Fields: map[string]any{"message": "test2"}}},
		},
		result: &LogBatchResult{},
	}

	events := batch.Events()

	assert.Len(t, events, 2)
	assert.Equal(t, batch.pendingEvents, events)
}

func TestLogBatchACK(t *testing.T) {
	batch := &LogBatch{
		result: &LogBatchResult{},
	}

	batch.ACK()

	assert.True(t, batch.result.Acked)
	assert.False(t, batch.result.Dropped)
	assert.False(t, batch.result.Retry)
	assert.False(t, batch.result.Split)
	assert.False(t, batch.result.Cancelled)
}

func TestLogBatchDrop(t *testing.T) {
	batch := &LogBatch{
		result: &LogBatchResult{},
	}

	batch.Drop()

	assert.True(t, batch.result.Dropped)
	assert.False(t, batch.result.Acked)
	assert.False(t, batch.result.Retry)
	assert.False(t, batch.result.Split)
	assert.False(t, batch.result.Cancelled)
}

func TestLogBatchRetry(t *testing.T) {
	batch := &LogBatch{
		result: &LogBatchResult{},
	}

	// First retry
	batch.Retry()
	assert.True(t, batch.result.Retry)
	assert.Equal(t, 1, batch.result.Retries)

	// Second retry
	batch.Retry()
	assert.True(t, batch.result.Retry)
	assert.Equal(t, 2, batch.result.Retries)
}

func TestLogBatchRetryEvents(t *testing.T) {
	events := []publisher.Event{
		{Content: beat.Event{Fields: map[string]any{"message": "test1"}}},
		{Content: beat.Event{Fields: map[string]any{"message": "test2"}}},
	}

	batch := &LogBatch{
		pendingEvents: events,
		result:        &LogBatchResult{},
	}

	retryEvents := []publisher.Event{events[1]}
	batch.RetryEvents(retryEvents)

	assert.Equal(t, retryEvents, batch.pendingEvents)
	assert.True(t, batch.result.Retry)
	assert.Equal(t, 1, batch.result.Retries)
}

func TestLogBatchSplitRetry(t *testing.T) {
	tests := []struct {
		name       string
		eventCount int
		wantSplit  bool
	}{
		{
			name:       "enough events to split",
			eventCount: 3,
			wantSplit:  true,
		},
		{
			name:       "minimum events to split",
			eventCount: 2,
			wantSplit:  true,
		},
		{
			name:       "not enough events to split - one event",
			eventCount: 1,
			wantSplit:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch := &LogBatch{
				pendingEvents: make([]publisher.Event, tt.eventCount),
				result:        &LogBatchResult{},
			}

			canSplit := batch.SplitRetry()

			assert.Equal(t, tt.wantSplit, canSplit)
			assert.Equal(t, tt.wantSplit, batch.result.Split)
		})
	}
}

func TestLogBatchCancelled(t *testing.T) {
	batch := &LogBatch{
		result: &LogBatchResult{},
	}

	batch.Cancelled()

	assert.True(t, batch.result.Cancelled)
	assert.False(t, batch.result.Acked)
	assert.False(t, batch.result.Dropped)
	assert.False(t, batch.result.Retry)
	assert.False(t, batch.result.Split)
}

func TestLogBatchResult(t *testing.T) {
	expectedResult := &LogBatchResult{
		Acked:     true,
		Dropped:   false,
		Retry:     true,
		Split:     false,
		Cancelled: false,
		Retries:   2,
	}

	batch := &LogBatch{
		result: expectedResult,
	}

	result := batch.Result()

	assert.Equal(t, expectedResult, result)
}

func mustParseTime(timeStr string) time.Time {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		panic(err)
	}
	return t
}

func countLogRecords(logs plog.Logs) int {
	count := 0
	for _, rl := range logs.ResourceLogs().All() {
		for _, sl := range rl.ScopeLogs().All() {
			count += sl.LogRecords().Len()
		}
	}
	return count
}
