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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/otel/otelctx"
	"github.com/elastic/beats/v7/libbeat/otel/otelmap"
	"github.com/elastic/beats/v7/libbeat/outputs"
	_ "github.com/elastic/beats/v7/libbeat/outputs/elasticsearch" // register "elasticsearch" output type
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	"github.com/elastic/beats/v7/libbeat/publisher"
	agentconfig "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	mockesapi "github.com/elastic/mock-es/pkg/api"
)

func TestPublish(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	event1 := beat.Event{Fields: mapstr.M{"field": 1}}
	event2 := beat.Event{Fields: mapstr.M{"field": 2}}
	event3 := beat.Event{Fields: mapstr.M{"field": 3}}
	event4 := beat.Event{Meta: mapstr.M{"_id": "abc123"}}

	beatInfo := beat.Info{Name: "testbeat", Version: "0.0.0"}

	makeOtelConsumer := func(t *testing.T, consumeFn func(ctx context.Context, ld plog.Logs) error) *otelConsumer {
		t.Helper()

		logger := logptest.NewTestingLogger(t, "")
		logConsumer, err := consumer.NewLogs(consumeFn)
		assert.NoError(t, err)
		return &otelConsumer{
			observer:     outputs.NewNilObserver(),
			logsConsumer: logConsumer,
			beatInfo:     beatInfo,
			log:          logger.Named("otelconsumer"),
			retry:        retryConfig{init: 1 * time.Millisecond, max: 2 * time.Millisecond},
		}
	}

	t.Run("ack batch on consumer success", func(t *testing.T) {
		batch := outest.NewBatch(event1, event2, event3)

		var countLogs int
		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			countLogs = countLogs + ld.LogRecordCount()
			return nil
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.NoError(t, err)
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchACK, batch.Signals[0].Tag)
		assert.Equal(t, len(batch.Events()), countLogs, "all events should be consumed")
	})

	t.Run("batches with errors report correct active event count", func(t *testing.T) {
		blockChan := make(chan struct{})
		defer close(blockChan)
		publishDone := make(chan struct{})
		batch := outest.NewBatch(event1, event2, event3)
		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			// Read from the channel twice: once to synchronize with the testing code so
			// we know the Publish call is waiting on the consume callback, then once
			// more to unblock it and allow Publish to resume.
			<-blockChan
			<-blockChan
			return fmt.Errorf("Some kind of error")
		})
		reg := monitoring.NewRegistry()
		otelConsumer.observer = outputs.NewStats(reg, logptest.NewTestingLogger(t, "testing"))
		assert.EqualValues(t, 0, checkEventsActive(reg), "initial total events should be zero")
		// Run Publish asynchronously so we can check the metrics while it is still in progress
		go func() {
			_ = otelConsumer.Publish(ctx, batch)
			// Signal that Publish has completed
			publishDone <- struct{}{}
		}()

		// Wait until Publish has called consume
		blockChan <- struct{}{}
		assert.EqualValues(t, 3, checkEventsActive(reg), "total event count should be 3 while Publish is waiting on downstream consumer")

		// Allow Publish to resume, and wait for it to finish
		blockChan <- struct{}{}
		<-publishDone

		assert.EqualValues(t, 0, checkEventsActive(reg), "final total events should be zero")
	})

	t.Run("data_stream fields are set on logrecord.Attribute", func(t *testing.T) {
		want := map[string]any{
			"type":      "logs",
			"namespace": "not_default",
			"dataset":   "not_elastic_agent",
		}
		for _, tc := range []struct {
			name  string
			value any
		}{
			{"mapstr.M", mapstr.M(want)},
			{"map[string]any", want},
		} {
			t.Run(tc.name, func(t *testing.T) {
				event1.Fields["data_stream"] = tc.value

				var attributes pcommon.Map
				oc := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
					for _, rl := range ld.ResourceLogs().All() {
						for _, sl := range rl.ScopeLogs().All() {
							for _, record := range sl.LogRecords().All() {
								attributes = record.Attributes()
							}
						}
					}
					return nil
				})

				require.NoError(t, oc.Publish(ctx, outest.NewBatch(event1)))
				for _, sub := range []string{"dataset", "namespace", "type"} {
					gotValue, ok := attributes.Get("data_stream." + sub)
					require.True(t, ok, "data_stream.%s not found on log record attribute", sub)
					assert.EqualValues(t, want[sub], gotValue.AsRaw())
				}
			})
		}
	})

	t.Run("Test elasticsearch.ingest_pipeline and elastic.mapping.mode fields are set", func(t *testing.T) {
		event1.Meta = mapstr.M{}
		event1.Meta["pipeline"] = "error_pipeline"

		batch := outest.NewBatch(event1)

		var countLogs int
		var scopeAttributes pcommon.Map
		var attributes pcommon.Map
		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			countLogs = countLogs + ld.LogRecordCount()
			for i := 0; i < ld.ResourceLogs().Len(); i++ {
				resourceLog := ld.ResourceLogs().At(i)
				for j := 0; j < resourceLog.ScopeLogs().Len(); j++ {
					scopeLog := resourceLog.ScopeLogs().At(j)
					scopeAttributes = scopeLog.Scope().Attributes()
					for k := 0; k < scopeLog.LogRecords().Len(); k++ {
						LogRecord := scopeLog.LogRecords().At(k)
						attributes = LogRecord.Attributes()
					}
				}
			}
			return nil
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.NoError(t, err)
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchACK, batch.Signals[0].Tag)

		dynamicAttributeKey := "elasticsearch.ingest_pipeline"
		gotValue, ok := attributes.Get(dynamicAttributeKey)
		require.True(t, ok, "dynamic pipeline attribute was not set")
		assert.Equal(t, "error_pipeline", gotValue.AsString())

		dynamicAttributeKey = "elastic.mapping.mode"
		gotValue, ok = scopeAttributes.Get(dynamicAttributeKey)
		require.True(t, ok, "elastic mapping mode was not set")
		assert.Equal(t, "bodymap", gotValue.AsString())
	})

	t.Run("preserves time.Duration fields as nanoseconds", func(t *testing.T) {
		eventWithDuration := beat.Event{
			Fields: mapstr.M{
				"event": mapstr.M{
					"duration": 1500 * time.Millisecond,
				},
			},
		}

		batch := outest.NewBatch(eventWithDuration)
		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			record := ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
			body := record.Body().Map().AsRaw()
			eventBody, ok := body["event"].(map[string]any)
			require.True(t, ok, "event body should be encoded as a map")
			assert.EqualValues(t, 1500*time.Millisecond, eventBody["duration"])
			return nil
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.NoError(t, err)
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchACK, batch.Signals[0].Tag)
	})

	t.Run("retries the batch on non-permanent consumer error", func(t *testing.T) {
		batch := outest.NewBatch(event1, event2, event3)

		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			return errors.New("consume error")
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.NoError(t, err)
		assert.False(t, consumererror.IsPermanent(err))
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchRetry, batch.Signals[0].Tag)
	})

	t.Run("drop batch on permanent consumer error", func(t *testing.T) {
		batch := outest.NewBatch(event1, event2, event3)

		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			return consumererror.NewPermanent(errors.New("consumer error"))
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.NoError(t, err)
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchDrop, batch.Signals[0].Tag)
	})

	t.Run("retries on context cancelled", func(t *testing.T) {
		batch := outest.NewBatch(event1, event2, event3)

		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			return context.Canceled
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.NoError(t, err)
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchRetry, batch.Signals[0].Tag)
	})

	t.Run("retryable errors wait a per-attempt jittered backoff", func(t *testing.T) {
		const initBackoff = 50 * time.Millisecond

		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			return errors.New("retryable error")
		})
		otelConsumer.retry = retryConfig{init: initBackoff, max: 500 * time.Millisecond}

		const margin = 25 * time.Millisecond
		for range 4 {
			batch := outest.NewBatch(event1)
			start := time.Now()
			err := otelConsumer.Publish(ctx, batch)
			d := time.Since(start)
			require.NoError(t, err, "Publish should not return an error")
			assert.Equal(t, outest.BatchRetry, batch.Signals[0].Tag, "batch should be retried")
			assert.GreaterOrEqual(t, d, initBackoff, "retry delay should be at least init")
			assert.Less(t, d, 2*initBackoff+margin, "retry delay should not escalate past ~2*init (got %v)", d)
		}
	})

	t.Run("cancels batch when context is cancelled during backoff", func(t *testing.T) {
		batch := outest.NewBatch(event1)

		cancelCtx, cancelFn := context.WithCancel(context.Background())
		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			return errors.New("retryable error")
		})
		otelConsumer.retry = retryConfig{init: 10 * time.Second, max: 10 * time.Second}

		publishDone := make(chan struct{})
		go func() {
			_ = otelConsumer.Publish(cancelCtx, batch)
			close(publishDone)
		}()

		time.Sleep(50 * time.Millisecond)
		cancelFn()
		<-publishDone

		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchCancelled, batch.Signals[0].Tag)
	})

	t.Run("sets the elasticsearchexporter doc id attribute from metadata", func(t *testing.T) {
		batch := outest.NewBatch(event4)

		var docID string
		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			record := ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
			attr, ok := record.Attributes().Get(esDocumentIDAttribute)
			assert.True(t, ok, "document ID attribute should be set")
			docID = attr.AsString()

			return nil
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.NoError(t, err)
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchACK, batch.Signals[0].Tag)
		assert.Equal(t, event4.Meta["_id"], docID)
	})

	t.Run("sets the receivertest doc id attribute", func(t *testing.T) {
		batch := outest.NewBatch(event4)

		var receivertestID string
		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			record := ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
			attr, ok := record.Attributes().Get(receivertest.UniqueIDAttrName)
			require.True(t, ok, "document ID attribute should be set")
			receivertestID = attr.AsString()

			return nil
		})
		otelConsumer.isReceiverTest = true

		err := otelConsumer.Publish(ctx, batch)
		assert.NoError(t, err)
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchACK, batch.Signals[0].Tag)
		assert.Equal(t, event4.Meta["_id"], receivertestID, "receivertest ID should match the event ID")
	})

	t.Run("sets the @timestamp field with the correct format", func(t *testing.T) {
		batch := outest.NewBatch(event3)
		batch.Events()[0].Content.Timestamp = time.Date(2025, time.January, 29, 9, 2, 39, 0, time.UTC)

		var bodyTimestamp string
		var recordTimestamp string
		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			record := ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
			field, ok := record.Body().Map().Get("@timestamp")
			recordTimestamp = record.Timestamp().AsTime().UTC().Format("2006-01-02T15:04:05.000Z")
			assert.True(t, ok, "timestamp field not found")
			bodyTimestamp = field.AsString()
			return nil
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.NoError(t, err)
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchACK, batch.Signals[0].Tag)
		assert.Equal(t, bodyTimestamp, recordTimestamp, "log record timestamp should match body timestamp")
	})

	t.Run("sets observed timestamp with the correct format", func(t *testing.T) {
		eventTime := time.Date(2025, time.January, 29, 9, 2, 39, 0, time.UTC)
		eventCreatedTime := eventTime.Add(-time.Minute)

		eventWithTime := beat.Event{Fields: mapstr.M{"event": mapstr.M{"created": eventCreatedTime}}}
		eventWithInvalidTime := beat.Event{Fields: mapstr.M{"event": mapstr.M{"created": 42}}}
		events := []beat.Event{event1, eventWithTime, eventWithInvalidTime}
		batch := outest.NewBatch(events...)
		for _, ev := range batch.Events() {
			ev.Content.Timestamp = eventTime
		}

		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			logRecords := ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords()
			assert.Equal(t, len(events), logRecords.Len(), "log records should be equal to events in the batch")

			// no event.created, observed timestamp should be the same as the event timestamp
			record := logRecords.At(0)
			recordTimestamp := record.Timestamp().AsTime().UTC().Format("2006-01-02T15:04:05.000Z")
			observedTimestamp := record.ObservedTimestamp().AsTime().UTC().Format("2006-01-02T15:04:05.000Z")
			assert.Equal(t, recordTimestamp, observedTimestamp, "observed timestamp should match event timestamp")

			// has event.created, observed timestamp should be the same as event.created
			record = logRecords.At(1)
			observedTimestamp = record.ObservedTimestamp().AsTime().UTC().Format("2006-01-02T15:04:05.000Z")
			eventCreatedTimestamp := eventCreatedTime.UTC().Format("2006-01-02T15:04:05.000Z")
			assert.Equal(t, eventCreatedTimestamp, observedTimestamp, "observed timestamp should match event.created")

			// has event.created with invalid type, observed timestamp should fall back to the event timestamp
			record = logRecords.At(2)
			recordTimestamp = record.Timestamp().AsTime().UTC().Format("2006-01-02T15:04:05.000Z")
			observedTimestamp = record.ObservedTimestamp().AsTime().UTC().Format("2006-01-02T15:04:05.000Z")
			assert.Equal(t, recordTimestamp, observedTimestamp, "observed timestamp should match log record timestamp")
			return nil
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.NoError(t, err)
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchACK, batch.Signals[0].Tag)
	})
	t.Run("sets the client context metadata with the beat info", func(t *testing.T) {
		batch := outest.NewBatch(event1)
		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			cm := client.FromContext(ctx).Metadata
			assert.Equal(t, beatInfo.Beat, cm.Get(otelctx.BeatNameCtxKey)[0])
			assert.Equal(t, beatInfo.Version, cm.Get(otelctx.BeatVersionCtxKey)[0])
			return nil
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.NoError(t, err)
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchACK, batch.Signals[0].Tag)
	})

	t.Run("includes metadata in the body without mutating the source event", func(t *testing.T) {
		beatInfo.IncludeMetadata = true

		eventWithMetadata := beat.Event{
			Meta: mapstr.M{
				"raw_index": "logs-test",
				"input_id":  "input-123",
			},
			Fields: mapstr.M{
				"message": "hello world",
			},
		}

		batch := outest.NewBatch(eventWithMetadata)
		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			record := ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
			body := record.Body().Map().AsRaw()

			metadata, ok := body["@metadata"].(map[string]any)
			require.True(t, ok, "@metadata should be present in the log body")
			assert.Equal(t, "logs-test", metadata["raw_index"])
			assert.Equal(t, "input-123", metadata["input_id"])
			assert.Equal(t, beatInfo.Beat, metadata["beat"])
			assert.Equal(t, beatInfo.Version, metadata["version"])
			assert.Equal(t, "_doc", metadata["type"])
			return nil
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.NoError(t, err)
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchACK, batch.Signals[0].Tag)
		assert.Equal(t, mapstr.M{"message": "hello world"}, eventWithMetadata.Fields)
		assert.Equal(t, mapstr.M{"raw_index": "logs-test", "input_id": "input-123"}, eventWithMetadata.Meta)
	})

	t.Run("includes metadata with no source meta fields", func(t *testing.T) {
		beatInfo.IncludeMetadata = true

		eventNoMeta := beat.Event{
			Fields: mapstr.M{"message": "hello"},
		}

		batch := outest.NewBatch(eventNoMeta)
		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			record := ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
			body := record.Body().Map().AsRaw()

			metadata, ok := body["@metadata"].(map[string]any)
			require.True(t, ok, "@metadata should be present even with nil source meta")
			assert.Equal(t, beatInfo.Beat, metadata["beat"])
			assert.Equal(t, beatInfo.Version, metadata["version"])
			assert.Equal(t, "_doc", metadata["type"])
			return nil
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.NoError(t, err)
	})
}

func checkEventsActive(reg *monitoring.Registry) int64 {
	outputSnapshot := monitoring.CollectFlatSnapshot(reg, monitoring.Full, true)
	return outputSnapshot.Ints["events.active"]
}

func TestFillLogRecordFromEventDoesNotError(t *testing.T) {
	logger := logp.NewNopLogger()
	beatInfo := beat.Info{}

	for _, tc := range benchmarkEventCases() {
		t.Run(tc.name, func(t *testing.T) {
			pubEvent := publisher.Event{Content: tc.event}
			logRecord := plog.NewLogRecord()
			if err := fillLogRecordFromEvent(logRecord, pubEvent, beatInfo, logger, false); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestElasticsearchOutputVsExporterSerialization verifies that Beat events are
// serialized identically across three paths for every fixture in
// otelmap.BenchmarkCases:
//   - Beats Elasticsearch output (go-structform encoder)
//   - OTel path: otelmap.FromMapstr (direct) -> ES exporter (bodymap)
func TestElasticsearchOutputVsExporterSerialization(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Build the Beats ES output once and reuse it across all sub-tests.
	beatsDocCh := make(chan []byte, 1)
	beatsSrv := httptest.NewServer(newMockES(t, func(_ mockesapi.Action, event []byte) int {
		beatsDocCh <- event
		return http.StatusOK
	}))
	t.Cleanup(beatsSrv.Close)

	beatsGroup, err := outputs.Load(
		testIndexManager{},
		beat.Info{Name: "testbeat", Version: "0.0.0", Logger: logger},
		nil,
		"elasticsearch",
		agentconfig.MustNewConfigFrom(mapstr.M{"hosts": []string{beatsSrv.URL}}),
	)
	require.NoError(t, err)
	require.Len(t, beatsGroup.Clients, 1)
	beatsClient, ok := beatsGroup.Clients[0].(outputs.NetworkClient)
	require.True(t, ok, "ES output client must implement outputs.NetworkClient")
	beatsEnc := beatsGroup.EncoderFactory()

	setupCtx, setupCancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer setupCancel()
	require.NoError(t, beatsClient.Connect(setupCtx))

	for _, tc := range otelmap.BenchmarkCases() {
		t.Run(tc.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()

			// Strip @timestamp from the fields so both Beats and OTel use
			// beatEvent.Timestamp as the canonical source, avoiding a conflict
			// when the bench case has its own @timestamp field.
			fields := tc.Src.Clone()
			delete(fields, "@timestamp")
			beatEvent := beat.Event{Timestamp: timestamp, Fields: fields}

			// ── OTel path: via otelConsumer.Publish (production code path) ────
			otelDoc := collectOtelDocViaPublish(t, ctx, logger, beatEvent)

			// ── Beats ES output path ──────────────────────────────────────────
			beatsBatch := outest.NewBatch(beatEvent)
			encodedEvent, encodedSize := beatsEnc.EncodeEntry(beatsBatch.Events()[0])
			if encodedSize == 0 {
				t.Logf("skipping Beats comparison for case %q: EncodeEntry produced no output — Src contains a type unsupported by go-structform", tc.Name)
				return
			}
			beatsBatch.Events()[0] = encodedEvent
			require.NoError(t, beatsClient.Publish(ctx, beatsBatch))

			var beatsDoc []byte
			select {
			case beatsDoc = <-beatsDocCh:
			case <-ctx.Done():
				t.Fatal("timed out waiting for Beats ES output to deliver document")
			}
			t.Run("beats_vs_otel", func(t *testing.T) {
				compareJSONValues(t, "beats", "otel", beatsDoc, otelDoc)
			})
		})
	}
}

// newTestESConsumer creates a fresh mock ES server backed by an ES exporter and
// returns a consumer.Logs that forwards to it plus a channel that receives each
// raw JSON document the mock server captures. Both are registered for cleanup on t.
func newTestESConsumer(t *testing.T, ctx context.Context) (consumer.Logs, <-chan []byte) {
	t.Helper()

	docCh := make(chan []byte, 1)
	srv := httptest.NewServer(newMockES(t, func(_ mockesapi.Action, event []byte) int {
		docCh <- event
		return http.StatusOK
	}))
	t.Cleanup(srv.Close)

	f := elasticsearchexporter.NewFactory()
	cfg, ok := f.CreateDefaultConfig().(*elasticsearchexporter.Config)
	require.Truef(t, ok, "elasticsearchexporter config must be *elasticsearchexporter.Config")
	cfg.Endpoints = []string{srv.URL}
	qb := cfg.QueueBatchConfig.Get()
	qb.NumConsumers = 1
	qb.Batch.Get().FlushTimeout = 50 * time.Millisecond

	esExp, err := f.CreateLogs(ctx, exportertest.NewNopSettings(f.Type()), cfg)
	require.NoError(t, err)
	require.NoError(t, esExp.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() { _ = esExp.Shutdown(context.Background()) })

	logConsumer, err := consumer.NewLogs(func(ctx context.Context, ld plog.Logs) error {
		return esExp.ConsumeLogs(ctx, ld)
	})
	require.NoError(t, err)
	return logConsumer, docCh
}

// collectOtelDocViaPublish sends beatEvent through the production
// otelConsumer.Publish path and returns the raw JSON document captured by the
// mock ES server.
func collectOtelDocViaPublish(t *testing.T, ctx context.Context, logger *logp.Logger, beatEvent beat.Event) []byte {
	t.Helper()
	logConsumer, docCh := newTestESConsumer(t, ctx)
	oc := &otelConsumer{
		observer:     outputs.NewNilObserver(),
		logsConsumer: logConsumer,
		beatInfo:     beat.Info{Name: "testbeat", Version: "0.0.0"},
		log:          logger.Named("otelconsumer"),
		retry:        retryConfig{init: 1 * time.Millisecond, max: 2 * time.Millisecond},
	}
	require.NoError(t, oc.Publish(ctx, outest.NewBatch(beatEvent)))
	select {
	case doc := <-docCh:
		return doc
	case <-ctx.Done():
		t.Fatal("timed out waiting for OTel exporter to deliver document to mock server")
		return nil
	}
}

// compareJSONValues compares two JSON documents for exact equality, preserving
// number token representation so "2" and "2.0" are distinguishable, while
// tolerating non-deterministic JSON object key ordering.
//
// Decoding with UseNumber stores numbers as json.Number (a string type), so
// the cmp.Comparer below compares them by their raw text, keeping "2" ≠ "2.0".
func compareJSONValues(t *testing.T, nameA, nameB string, docA, docB []byte) {
	t.Helper()
	a := parseJSONDoc(t, nameA, docA)
	b := parseJSONDoc(t, nameB, docB)
	if diff := cmp.Diff(a, b, cmp.Comparer(func(x, y json.Number) bool {
		return x.String() == y.String()
	})); diff != "" {
		t.Errorf("%s vs %s differ (-want +got):\n%s", nameA, nameB, diff)
	}
}

func parseJSONDoc(t *testing.T, name string, data []byte) any {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	require.NoErrorf(t, dec.Decode(&v), "parse %s JSON: %s", name, data)
	return v
}

// newMockES creates a mock-es APIHandler that calls handler for each document
// in every bulk request.
func newMockES(t *testing.T, handler func(mockesapi.Action, []byte) int) *mockesapi.APIHandler {
	t.Helper()
	return mockesapi.NewDeterministicAPIHandler(
		uuid.Must(uuid.NewV4()),
		"",
		nil,
		time.Now().Add(time.Hour),
		0,
		10,
		handler,
	)
}

func TestSanitizeDataStreamField(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		disallowed string
		want       string
	}{
		{
			name:       "clean value unchanged",
			input:      "http",
			disallowed: `-\/*?"<>| ,#:`,
			want:       "http",
		},
		{
			name:       "uppercase lowercased",
			input:      "HTTP",
			disallowed: `-\/*?"<>| ,#:`,
			want:       "http",
		},
		{
			name:       "disallowed rune replaced with underscore",
			input:      "my dataset",
			disallowed: `-\/*?"<>| ,#:`,
			want:       "my_dataset",
		},
		{
			name:       "multiple disallowed runes replaced",
			input:      "a*b?c",
			disallowed: `-\/*?"<>| ,#:`,
			want:       "a_b_c",
		},
		{
			name:       "value truncated to maxDataStreamBytes",
			input:      string(make([]byte, 110)),
			disallowed: `-\/*?"<>| ,#:`,
			want:       string(make([]byte, 100)),
		},
		{
			name:       "namespace allows hyphen, dataset does not",
			input:      "my-namespace",
			disallowed: `\/*?"<>| ,#:`,
			want:       "my-namespace",
		},
		{
			name:       "dataset treats hyphen as disallowed",
			input:      "my-dataset",
			disallowed: `-\/*?"<>| ,#:`,
			want:       "my_dataset",
		},
	}
	const maxLength = 100
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeDataStreamField(tc.input, tc.disallowed, maxLength)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFillLogRecordSetsElasticsearchIndex(t *testing.T) {
	logger := logp.NewNopLogger()
	beatInfo := beat.Info{}

	cases := []struct {
		name      string
		dsType    string
		dataset   string
		namespace string
		wantIndex string // empty means attribute should not be set
	}{
		{
			name:      "synthetics type sets elasticsearch.index",
			dsType:    "synthetics",
			dataset:   "http",
			namespace: "default",
			wantIndex: "synthetics-http-default",
		},
		{
			name:      "traces type sets elasticsearch.index",
			dsType:    "traces",
			dataset:   "apm.transaction",
			namespace: "default",
			wantIndex: "traces-apm.transaction-default",
		},
		{
			name:      "logs type does not set elasticsearch.index",
			dsType:    "logs",
			dataset:   "system.syslog",
			namespace: "default",
			wantIndex: "",
		},
		{
			name:      "metrics type does not set elasticsearch.index",
			dsType:    "metrics",
			dataset:   "system.cpu",
			namespace: "default",
			wantIndex: "",
		},
		{
			name:      "synthetics dataset sanitized",
			dsType:    "synthetics",
			dataset:   "http monitor",
			namespace: "my-namespace",
			wantIndex: "synthetics-http_monitor-my-namespace",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			event := publisher.Event{
				Content: beat.Event{
					Fields: mapstr.M{
						"data_stream": mapstr.M{
							"type":      tc.dsType,
							"dataset":   tc.dataset,
							"namespace": tc.namespace,
						},
					},
				},
			}
			logRecord := plog.NewLogRecord()
			err := fillLogRecordFromEvent(logRecord, event, beatInfo, logger, false)
			require.NoError(t, err)

			indexAttr, hasIndex := logRecord.Attributes().Get("elasticsearch.index")
			if tc.wantIndex == "" {
				assert.False(t, hasIndex, "elasticsearch.index should not be set for type %q", tc.dsType)
			} else {
				require.True(t, hasIndex, "elasticsearch.index should be set for type %q", tc.dsType)
				assert.Equal(t, tc.wantIndex, indexAttr.Str())
			}
		})
	}
}

// testIndexManager is a minimal outputs.IndexManager that always selects a fixed index name.
type testIndexManager struct{}

func (testIndexManager) BuildSelector(_ *agentconfig.C) (outputs.IndexSelector, error) {
	return testIndexSelector{}, nil
}

type testIndexSelector struct{}

func (testIndexSelector) Select(_ *beat.Event) (string, error) {
	return "test-index", nil
}
