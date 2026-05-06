// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package otelconsumer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
)

type benchmarkEventCase struct {
	name  string
	event beat.Event
}

func benchmarkEventCases() []benchmarkEventCase {
	timestamp := time.Date(2026, 4, 23, 12, 34, 56, 789000000, time.UTC)
	created := timestamp.Add(-1500 * time.Millisecond)

	return []benchmarkEventCase{
		{
			name: "minimal",
			event: beat.Event{
				Timestamp: timestamp,
				Fields: mapstr.M{
					"message": "hello world",
				},
			},
		},
		{
			name: "ecs_log",
			event: beat.Event{
				Timestamp: timestamp,
				Meta: mapstr.M{
					"_id":      "abc123",
					"pipeline": "logs-default-pipeline",
				},
				Fields: mapstr.M{
					"message": "GET /api/orders 200 12ms",
					"event": mapstr.M{
						"dataset": "service.access",
						"created": created,
					},
					"log": mapstr.M{
						"level": "info",
						"file": mapstr.M{
							"path": "/var/log/service/access.log",
						},
					},
					"host": mapstr.M{
						"name": "edge-01",
						"ip":   []string{"10.0.0.10", "192.168.10.5"},
					},
					"service": mapstr.M{
						"name":    "checkout",
						"version": "1.2.3",
					},
					"data_stream": mapstr.M{
						"type":      "logs",
						"dataset":   "service.access",
						"namespace": "prod",
					},
				},
			},
		},
		{
			name: "nested_nonprimitive",
			event: beat.Event{
				Timestamp: timestamp,
				Fields: mapstr.M{
					"event": mapstr.M{
						"created": common.Time(created),
					},
					"labels": []string{"prod", "payments", "blue"},
					"durations": []time.Time{
						created,
						timestamp,
					},
					"nested": []mapstr.M{
						{
							"id":   1,
							"tags": []string{"a", "b", "c"},
						},
						{
							"id": 2,
							"meta": mapstr.M{
								"enabled": true,
								"codes":   []int{200, 201, 202},
							},
						},
					},
				},
			},
		},
	}
}

func makeTestOtelConsumer(t testing.TB, consumeFn func(ctx context.Context, ld plog.Logs) error) *otelConsumer {
	t.Helper()

	logConsumer, err := consumer.NewLogs(consumeFn)
	assert.NoError(t, err)
	consumer := &otelConsumer{
		observer:     outputs.NewNilObserver(),
		logsConsumer: logConsumer,
		beatInfo:     beat.Info{},
		log:          logp.NewNopLogger(),
	}
	return consumer
}

func makeBenchmarkBatch(event beat.Event, size int) []beat.Event {
	events := make([]beat.Event, size)
	for i := range size {
		events[i] = event
	}
	return events
}

func fillLogRecordFromEventFromRaw(logRecord plog.LogRecord, event publisher.Event, beatInfo beat.Info, log *logp.Logger, isReceiverTest bool) error {
	beatEvent := prepareLogRecordFromEvent(logRecord, event, log, isReceiverTest)
	return encodeLogRecordBodyFromRawWithTimestamp(logRecord, beatEvent, event.Content.Timestamp, logBodyMetadata(event, beatInfo))
}

func buildLogsForEvents(events []beat.Event, beatInfo beat.Info, logger *logp.Logger, fillFn func(plog.LogRecord, publisher.Event, beat.Info, *logp.Logger, bool) error) (plog.Logs, error) {
	pLogs := plog.NewLogs()
	resourceLogs := pLogs.ResourceLogs().AppendEmpty()
	sourceLogs := resourceLogs.ScopeLogs().AppendEmpty()
	sourceLogs.Scope().Attributes().PutStr("elastic.mapping.mode", "bodymap")

	logRecords := sourceLogs.LogRecords()
	logRecords.EnsureCapacity(len(events))
	for _, event := range events {
		logRecord := logRecords.AppendEmpty()
		if err := fillFn(logRecord, publisher.Event{Content: event}, beatInfo, logger, false); err != nil {
			return plog.NewLogs(), fmt.Errorf("fill log record: %w", err)
		}
	}
	return pLogs, nil
}

func publishBatchWithFill(out *otelConsumer, ctx context.Context, batch publisher.Batch, fillFn func(plog.LogRecord, publisher.Event, beat.Info, *logp.Logger, bool) error) error {
	events := batch.Events()
	beatEvents := make([]beat.Event, len(events))
	for i, event := range events {
		beatEvents[i] = event.Content
	}
	pLogs, err := buildLogsForEvents(beatEvents, out.beatInfo, out.log, fillFn)
	if err != nil {
		return err
	}
	if err := out.logsConsumer.ConsumeLogs(ctx, pLogs); err != nil {
		return err
	}
	batch.ACK()
	return nil
}

func assertLogRecordsEquivalent(tb testing.TB, expected, actual plog.LogRecord) {
	tb.Helper()
	assert.Equal(tb, expected.Timestamp().AsTime(), actual.Timestamp().AsTime())
	assert.Equal(tb, expected.ObservedTimestamp().AsTime(), actual.ObservedTimestamp().AsTime())
	assert.Equal(tb, expected.Attributes().AsRaw(), actual.Attributes().AsRaw())
	assert.Equal(tb, expected.Body().AsRaw(), actual.Body().AsRaw())
}

func assertLogsEquivalent(tb testing.TB, expected, actual plog.Logs) {
	tb.Helper()

	assert.Equal(tb, expected.LogRecordCount(), actual.LogRecordCount())
	assert.Equal(tb, expected.ResourceLogs().Len(), actual.ResourceLogs().Len())

	for i := 0; i < expected.ResourceLogs().Len(); i++ {
		expectedResource := expected.ResourceLogs().At(i)
		actualResource := actual.ResourceLogs().At(i)
		assert.Equal(tb, expectedResource.ScopeLogs().Len(), actualResource.ScopeLogs().Len())
		for j := 0; j < expectedResource.ScopeLogs().Len(); j++ {
			expectedScope := expectedResource.ScopeLogs().At(j)
			actualScope := actualResource.ScopeLogs().At(j)
			assert.Equal(tb, expectedScope.Scope().Attributes().AsRaw(), actualScope.Scope().Attributes().AsRaw())
			assert.Equal(tb, expectedScope.LogRecords().Len(), actualScope.LogRecords().Len())
			for k := 0; k < expectedScope.LogRecords().Len(); k++ {
				assertLogRecordsEquivalent(tb, expectedScope.LogRecords().At(k), actualScope.LogRecords().At(k))
			}
		}
	}
}
