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
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
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
		{
			name: "packetbeat_flow_like",
			event: beat.Event{
				Timestamp: timestamp,
				Fields: mapstr.M{
					"event": mapstr.M{
						"start":    common.Time(timestamp.Add(-2 * time.Second)),
						"end":      common.Time(timestamp),
						"duration": 2 * time.Second,
						"category": []string{"network"},
						"type":     []string{"connection", "end"},
					},
					"flow": mapstr.M{
						"id":    common.NetString("flow-id"),
						"final": true,
						"vlan":  []uint64{100, 200},
					},
					"network": mapstr.M{
						"bytes":        uint64(1234),
						"packets":      uint64(12),
						"community_id": "1:abc",
					},
				},
			},
		},
		{
			name: "metricbeat_sql_like",
			event: beat.Event{
				Timestamp: timestamp,
				Fields: mapstr.M{
					"sql": mapstr.M{
						"row": mapstr.M{
							"string":         "000400",
							"unsigned_int":   uint64(100),
							"array":          []any{0, 1, 2},
							"byte_array":     "byte_array",
							"formatted_time": timestamp.Format(time.RFC3339Nano),
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
	for i := range events {
		events[i] = event
	}
	return events
}

// BenchmarkFillLogRecordFromEvent measures the cost of filling a single
// pdata log record from a beats event.
func BenchmarkFillLogRecordFromEvent(b *testing.B) {
	for _, tc := range benchmarkEventCases() {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			pubEvent := publisher.Event{Content: tc.event}
			logger := logp.NewNopLogger()
			beatInfo := beat.Info{}

			for b.Loop() {
				logRecord := plog.NewLogRecord()
				if err := fillLogRecordFromEvent(logRecord, pubEvent, beatInfo, logger, false); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkPublish exercises the full Publish flow with a no-op consumer,
// so the timer captures batching, log-record construction, and ConsumeLogs
// overhead together.
func BenchmarkPublish(b *testing.B) {
	for _, tc := range benchmarkEventCases() {
		for _, batchSize := range []int{1, 128, 1024} {
			b.Run(fmt.Sprintf("%s/%d_events", tc.name, batchSize), func(b *testing.B) {
				b.ReportAllocs()
				events := makeBenchmarkBatch(tc.event, batchSize)
				ctx := context.Background()

				var countLogs int
				otelConsumer := makeTestOtelConsumer(b, func(ctx context.Context, ld plog.Logs) error {
					countLogs += ld.LogRecordCount()
					return nil
				})

				for b.Loop() {
					batch := outest.NewBatch(events...)
					if err := otelConsumer.Publish(ctx, batch); err != nil {
						b.Fatal(err)
					}
					if len(batch.Signals) != 1 {
						b.Fatalf("expected 1 batch signal, got %d", len(batch.Signals))
					}
					if batch.Signals[0].Tag != outest.BatchACK {
						b.Fatalf("expected ACK batch signal, got %v", batch.Signals[0].Tag)
					}
				}
				b.StopTimer()

				assert.Equal(b, b.N*batchSize, countLogs, "all events should be consumed")
			})
		}
	}
}
