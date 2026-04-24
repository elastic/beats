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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/plog"
)

func BenchmarkFillLogRecordFromEvent(b *testing.B) {
	encoders := []struct {
		name string
		fn   func(plog.LogRecord, publisher.Event, beat.Info, *logp.Logger, bool) error
	}{
		{name: "from_raw", fn: fillLogRecordFromEventFromRaw},
		{name: "direct", fn: fillLogRecordFromEvent},
	}

	for _, encoder := range encoders {
		for _, tc := range benchmarkEventCases() {
			b.Run(fmt.Sprintf("%s/%s", encoder.name, tc.name), func(b *testing.B) {
				b.ReportAllocs()
				pubEvent := publisher.Event{Content: tc.event}
				logger := logp.NewNopLogger()
				beatInfo := beat.Info{}

				oracle := plog.NewLogRecord()
				candidate := plog.NewLogRecord()
				if err := fillLogRecordFromEventFromRaw(oracle, pubEvent, beatInfo, logger, false); err != nil {
					b.Fatal(err)
				}
				if err := encoder.fn(candidate, pubEvent, beatInfo, logger, false); err != nil {
					b.Fatal(err)
				}
				assertLogRecordsEquivalent(b, oracle, candidate)

				for b.Loop() {
					logRecord := plog.NewLogRecord()
					if err := encoder.fn(logRecord, pubEvent, beatInfo, logger, false); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func BenchmarkPublish(b *testing.B) {
	encoders := []struct {
		name string
		fn   func(plog.LogRecord, publisher.Event, beat.Info, *logp.Logger, bool) error
	}{
		{name: "from_raw", fn: fillLogRecordFromEventFromRaw},
		{name: "direct", fn: fillLogRecordFromEvent},
	}

	for _, encoder := range encoders {
		for _, tc := range benchmarkEventCases() {
			for _, batchSize := range []int{1, 128, 1024} {
				b.Run(fmt.Sprintf("%s/%s/%d_events", encoder.name, tc.name, batchSize), func(b *testing.B) {
					b.ReportAllocs()
					events := makeBenchmarkBatch(tc.event, batchSize)
					ctx := context.Background()

					var countLogs int
					otelConsumer := makeTestOtelConsumer(b, func(ctx context.Context, ld plog.Logs) error {
						countLogs += ld.LogRecordCount()
						return nil
					})

					oracleBatch := outest.NewBatch(events...)
					candidateBatch := outest.NewBatch(events...)
					if err := publishBatchWithFill(otelConsumer, ctx, oracleBatch, fillLogRecordFromEventFromRaw); err != nil {
						b.Fatal(err)
					}
					if err := publishBatchWithFill(otelConsumer, ctx, candidateBatch, encoder.fn); err != nil {
						b.Fatal(err)
					}
					if len(candidateBatch.Signals) != 1 {
						b.Fatalf("expected 1 batch signal, got %d", len(candidateBatch.Signals))
					}
					if candidateBatch.Signals[0].Tag != outest.BatchACK {
						b.Fatalf("expected ACK batch signal, got %v", candidateBatch.Signals[0].Tag)
					}

					b.ResetTimer()
					for b.Loop() {
						batch := outest.NewBatch(events...)
						if err := publishBatchWithFill(otelConsumer, ctx, batch, encoder.fn); err != nil {
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

					assert.Equal(b, 2*batchSize+b.N*batchSize, countLogs, "all events should be consumed")
				})
			}
		}
	}
}

func BenchmarkBuildLogs(b *testing.B) {
	encoders := []struct {
		name string
		fn   func(plog.LogRecord, publisher.Event, beat.Info, *logp.Logger, bool) error
	}{
		{name: "from_raw", fn: fillLogRecordFromEventFromRaw},
		{name: "direct", fn: fillLogRecordFromEvent},
	}

	for _, encoder := range encoders {
		for _, tc := range benchmarkEventCases() {
			for _, batchSize := range []int{1, 128, 1024} {
				b.Run(fmt.Sprintf("%s/%s/%d_events", encoder.name, tc.name, batchSize), func(b *testing.B) {
					b.ReportAllocs()
					events := makeBenchmarkBatch(tc.event, batchSize)
					logger := logp.NewNopLogger()
					beatInfo := beat.Info{}

					oracle, err := buildLogsForEvents(events, beatInfo, logger, fillLogRecordFromEventFromRaw)
					if err != nil {
						b.Fatal(err)
					}
					candidate, err := buildLogsForEvents(events, beatInfo, logger, encoder.fn)
					if err != nil {
						b.Fatal(err)
					}
					assertLogsEquivalent(b, oracle, candidate)

					for b.Loop() {
						if _, err := buildLogsForEvents(events, beatInfo, logger, encoder.fn); err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		}
	}
}
