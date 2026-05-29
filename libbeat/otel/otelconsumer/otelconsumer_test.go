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
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exportertest"

	"github.com/gofrs/uuid/v5"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/otel/otelctx"
	"github.com/elastic/beats/v7/libbeat/outputs"
	_ "github.com/elastic/beats/v7/libbeat/outputs/elasticsearch" // register "elasticsearch" output type
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	agentconfig "github.com/elastic/elastic-agent-libs/config"
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
		dataStreamField := mapstr.M{
			"type":      "logs",
			"namespace": "not_default",
			"dataset":   "not_elastic_agent",
		}
		event1.Fields["data_stream"] = dataStreamField

		batch := outest.NewBatch(event1)

		var countLogs int
		var attributes pcommon.Map
		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			countLogs = countLogs + ld.LogRecordCount()
			for i := 0; i < ld.ResourceLogs().Len(); i++ {
				resourceLog := ld.ResourceLogs().At(i)
				for j := 0; j < resourceLog.ScopeLogs().Len(); j++ {
					scopeLog := resourceLog.ScopeLogs().At(j)
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

		subFields := []string{"dataset", "namespace", "type"}
		for _, subField := range subFields {
			gotValue, ok := attributes.Get("data_stream." + subField)
			require.True(t, ok, "data_stream.%s not found on log record attribute", subField)
			assert.EqualValues(t, dataStreamField[subField], gotValue.AsRaw())
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

	t.Run("retries are delayed by exponential backoff", func(t *testing.T) {
		const (
			initBackoff = 50 * time.Millisecond
			maxBackoff  = 500 * time.Millisecond
		)

		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			return errors.New("retryable error")
		})
		otelConsumer.retry = retryConfig{init: initBackoff, max: maxBackoff}

		// Measure the duration of each Publish call. Each call blocks during
		// backoff Wait(), so elapsed time reflects the actual backoff delay.
		var durations []time.Duration
		for range 3 {
			batch := outest.NewBatch(event1)
			start := time.Now()
			err := otelConsumer.Publish(ctx, batch)
			durations = append(durations, time.Since(start))
			require.NoError(t, err, "Publish should not return an error")
			assert.Equal(t, outest.BatchRetry, batch.Signals[0].Tag, "batch should be retried")
		}

		assert.GreaterOrEqual(t, durations[0], initBackoff, "first retry delay should be at least ~init")
		assert.GreaterOrEqual(t, durations[1], 2*initBackoff, "second retry delay should be at least ~2*init (exponential growth)")
		assert.GreaterOrEqual(t, durations[2], 4*initBackoff, "third retry delay should be at least ~4*init (exponential growth)")
	})

	t.Run("backoff resets on success", func(t *testing.T) {
		const (
			initBackoff = 50 * time.Millisecond
			maxBackoff  = 500 * time.Millisecond
		)

		callCount := 0
		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			callCount++
			if callCount == 3 {
				return nil
			}
			return errors.New("retryable error")
		})
		otelConsumer.retry = retryConfig{init: initBackoff, max: maxBackoff}

		// Two failures grow the backoff past init level.
		batch1 := outest.NewBatch(event1)
		err := otelConsumer.Publish(ctx, batch1)
		require.NoError(t, err)
		assert.Equal(t, outest.BatchRetry, batch1.Signals[0].Tag, "first batch should be retried")

		batch2 := outest.NewBatch(event1)
		err = otelConsumer.Publish(ctx, batch2)
		require.NoError(t, err)
		assert.Equal(t, outest.BatchRetry, batch2.Signals[0].Tag, "second batch should be retried")

		// Third call succeeds, triggering backoff Reset().
		batch3 := outest.NewBatch(event1)
		err = otelConsumer.Publish(ctx, batch3)
		require.NoError(t, err)
		assert.Equal(t, outest.BatchACK, batch3.Signals[0].Tag, "third batch should be acked")

		// Next failure should use init-level backoff ([init, 2*init) = [50ms, 100ms)),
		// not the grown level which would be [4*init, 8*init) = [200ms, 400ms).
		batch4 := outest.NewBatch(event1)
		start := time.Now()
		err = otelConsumer.Publish(ctx, batch4)
		duration := time.Since(start)
		require.NoError(t, err)
		assert.Equal(t, outest.BatchRetry, batch4.Signals[0].Tag, "fourth batch should be retried")
		// In equal jitter backoff strategy, initial backoff is between initBackoff and 2*initBackoff.
		const margin = 10 * time.Millisecond
		assert.Less(t, duration, 2*initBackoff+margin, "after success, backoff should reset to init level, not continue growing (got %v)", duration)
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
}

func TestPublishRoutesToBatchSource(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger := logptest.NewTestingLogger(t, "")

	// collectFields returns a consumer.Logs that records the "field" value of
	// every log record it receives into the given slice.
	collectFields := func(received *[]int64) consumer.Logs {
		c, err := consumer.NewLogs(func(_ context.Context, ld plog.Logs) error {
			rl := ld.ResourceLogs()
			for i := 0; i < rl.Len(); i++ {
				sl := rl.At(i).ScopeLogs()
				for j := 0; j < sl.Len(); j++ {
					lr := sl.At(j).LogRecords()
					for k := 0; k < lr.Len(); k++ {
						if v, ok := lr.At(k).Body().Map().Get("field"); ok {
							*received = append(*received, v.Int())
						}
					}
				}
			}
			return nil
		})
		require.NoError(t, err)
		return c
	}

	// The pipeline splits batches by source, so the consumer always receives a
	// single-source batch and routes it to that source's consumer.
	t.Run("routes a single-source batch to that source's consumer", func(t *testing.T) {
		var received []int64
		source := beat.Info{Name: "beat1", LogConsumer: collectFields(&received)}
		own, err := consumer.NewLogs(func(context.Context, plog.Logs) error {
			require.Fail(t, "own consumer should not be used when the batch carries a Source")
			return nil
		})
		require.NoError(t, err)
		out := &otelConsumer{
			observer:     outputs.NewNilObserver(),
			logsConsumer: own,
			beatInfo:     beat.Info{Name: "own"},
			log:          logger.Named("otelconsumer"),
			retry:        retryConfig{init: time.Millisecond, max: 2 * time.Millisecond},
		}
		batch := outest.NewBatch(
			beat.Event{Fields: mapstr.M{"field": 1}},
			beat.Event{Fields: mapstr.M{"field": 2}},
		)
		for i := range batch.Events() {
			batch.Events()[i].Source = &source
		}
		require.NoError(t, out.Publish(ctx, batch))
		assert.ElementsMatch(t, []int64{1, 2}, received)
		require.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchACK, batch.Signals[0].Tag)
	})

	t.Run("routes an untagged batch to the consumer's own destination", func(t *testing.T) {
		var received []int64
		out := &otelConsumer{
			observer:     outputs.NewNilObserver(),
			logsConsumer: collectFields(&received),
			beatInfo:     beat.Info{Name: "own"},
			log:          logger.Named("otelconsumer"),
			retry:        retryConfig{init: time.Millisecond, max: 2 * time.Millisecond},
		}
		batch := outest.NewBatch(beat.Event{Fields: mapstr.M{"field": 7}})
		require.NoError(t, out.Publish(ctx, batch))
		assert.ElementsMatch(t, []int64{7}, received)
		require.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchACK, batch.Signals[0].Tag)
	})
}

func checkEventsActive(reg *monitoring.Registry) int64 {
	outputSnapshot := monitoring.CollectFlatSnapshot(reg, monitoring.Full, true)
	return outputSnapshot.Ints["events.active"]
}

// TestElasticsearchOutputVsExporterSerialization verifies that Beat events are serialized
// identically whether they flow through the Beats Elasticsearch output or
// through the OTel path (otelconsumer + ES exporter using bodymap mode).
func TestElasticsearchOutputVsExporterSerialization(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	fixedTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	beatEvent := beat.Event{
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Fields: mapstr.M{
			// ── Primitive types: signed integers (max and zero) ───────────────────
			"int_val":    int(42),
			"int8_val":   int8(math.MaxInt8),
			"int16_val":  int16(math.MaxInt16),
			"int32_val":  int32(math.MaxInt32),
			"int64_val":  int64(1000),
			"int_zero":   int(0),
			"int8_zero":  int8(0),
			"int16_zero": int16(0),
			"int32_zero": int32(0),
			"int64_zero": int64(0),

			// ── Primitive types: unsigned integers (max and zero) ─────────────────
			"uint_val":    uint(42),
			"uint8_val":   uint8(math.MaxUint8),
			"uint16_val":  uint16(math.MaxUint16),
			"uint32_val":  uint32(math.MaxUint32),
			"uint64_val":  uint64(1234),
			"uint_zero":   uint(0),
			"uint8_zero":  uint8(0),
			"uint16_zero": uint16(0),
			"uint32_zero": uint32(0),
			"uint64_zero": uint64(0),

			// byte = uint8 and rune = int32 — test via their named aliases
			"byte_val": byte('A'),
			"rune_val": rune('€'),

			// ── Primitive types: floats ───────────────────────────────────────────
			"float_val":     float64(1.5),
			"neg_float":     float64(-1.5),
			"float32_val":   float32(1.5),
			"float64_max":   math.MaxFloat64,
			"float64_large": math.MaxFloat64 / 3,

			// ── Primitive types: other scalars ────────────────────────────────────
			"bool_val":   true,
			"bool_false": false,
			"str_val":    "hello world",
			"str_empty":  "",
			"nil_val":    nil,

			// ── Collection types: signed integer slices (max, min, zero) ──────────
			"int_slice":   []int{1, 2, 3},
			"int8_slice":  []int8{math.MaxInt8, math.MinInt8, 0},
			"int16_slice": []int16{math.MaxInt16, math.MinInt16, 0},
			"int32_slice": []int32{math.MaxInt32, math.MinInt32, 0},
			"int64_slice": []int64{math.MaxInt64, math.MinInt64, 0},

			// ── Collection types: unsigned integer / bool / string slices ─────────
			"uint_slice":   []uint{0, 1, 2},
			"uint8_slice":  []uint8{0, 1, math.MaxUint8},
			"uint16_slice": []uint16{0, 1, math.MaxUint16},
			"uint32_slice": []uint32{0, 1, math.MaxUint32},
			"uint64_slice": []uint64{100, 200},
			"bool_slice":   []bool{true, false},
			"str_slice":    []string{"a", "b", "c"},
			"any_slice":    []any{1, "two", true},

			// ── Collection types: float slices ────────────────────────────────────

			"float64_slice": []float64{1.5, -2.5, 0.25},
			"float32_slice": []float32{1.5, -2.5, 0.25},

			// ── Time types ────────────────────────────────────────────────────────
			"time_slice":        []time.Time{fixedTime},
			"ts_field":          fixedTime,
			"duration_field":    1500 * time.Millisecond,
			"common_time_field": common.Time(fixedTime),
			"common_time_slice": []common.Time{common.Time(fixedTime)},

			// ── mapstr types ──────────────────────────────────────────────────────
			"mapstr_nested": mapstr.M{
				"str_field": "nested value",
				"int_field": int(7),
			},
			"mapstr_slice": []mapstr.M{
				{"id": int(1), "tag": "alpha"},
				{"id": int(2), "tag": "beta"},
			},

			// ── map[string]any (handled same as mapstr.M in ConvertNonPrimitive) ──
			"map_any": map[string]any{
				"str_field": "from map_any",
				"int_field": int(99),
			},

			// ── JSON types ────────────────────────────────────────────────────────
			// json.RawMessage is a named []byte type. Neither path passes through
			// the raw JSON: go-structform converts to []uint8 via liftFold and emits
			// each byte as an integer; ConvertNonPrimitive's generic slice branch
			// stores each byte as uint8, serialised by pcommon as an integer.
			// Both paths produce the same integer array (not a JSON pass-through).
			"json_raw": json.RawMessage(`{"key":"value"}`),

			// ── Known divergences (commented out) ─────────────────────────────────

			// TODO: NaN and Inf — go-structform's ES encoder has ignoreInvalidFloat=false
			// (unlike the codec JSON encoder). Encountering NaN or Inf returns
			// "unsupported float value: NaN" and aborts the Beats encoding entirely,
			// so no document is delivered to Elasticsearch and the test times out.
			// On the OTel side, ConvertNonPrimitive passes float64 through unchanged;
			// pcommon stores Double(NaN)/Double(Inf) and the ES exporter behaviour is
			// undefined (likely null or omitted field).
			// "float_nan":     math.NaN(),
			// "float_inf_pos": math.Inf(1),
			// "float_inf_neg": math.Inf(-1),

			// TODO: complex64 / complex128 — go-structform's getReflectFoldPrimitiveKind
			// returns errUnsupported for reflect.Complex64 and reflect.Complex128 (they
			// are absent from its generated kind switch), aborting the Beats encoding
			// with the same timeout failure as NaN/Inf above. On the OTel side,
			// ConvertNonPrimitive's default branch produces the string
			// "unknown type: complex64" / "unknown type: complex128".
			// "complex64_val":  complex64(1 + 2i),
			// "complex128_val": complex128(3 + 4i),

			// TODO(https://github.com/elastic/elastic-agent/issues/14610):
			// ExplicitRadixPoint=false (Beats) vs =true (OTel ES exporter) causes:
			//   • Decimal form: float64(2.0) → "2" (Beats) vs "2.0" (OTel).
			//   • Scientific-notation whole-number mantissa: math.SmallestNonzeroFloat64
			//     (5e-324) → "5e-324" (Beats) vs "5.0e-324" (OTel).
			// Affects scalars, nested map values, and slice elements.
			// "zero_float":   float64(0.0),
			// "float64_int":  float64(1.0),
			// "float32_int":  float32(2.0),
			// "float_slice":  []float64{1.5, 2.0, 0.0},
			// "float64_tiny": math.SmallestNonzeroFloat64,

			// TODO: common.NetString — go-structform encodes the underlying []byte
			// as a JSON integer array; the OTel path calls MarshalText() and stores
			// the string.
			// "net_string_field": common.NetString("hello"),

			// TODO: json.Number — ConvertNonPrimitive has no case for this named
			// string type; falls to "unknown type: json.Number". Beats go-structform
			// folds it as its underlying string value (e.g. json.Number("42") → "42").
			// "json_number": json.Number("42"),

			// TODO: [][]byte — go-structform serialises each inner []byte as a JSON
			// integer array; pcommon.Value.FromRaw([]byte) stores it as Bytes, which
			// the OTel ES exporter base64-encodes.
			// "bytes_slice": [][]byte{[]byte("hello"), []byte("world")},

			// TODO: []*conf.C — complex structured type; ConvertNonPrimitive falls
			// to the generic slice path which stores each element as interface{};
			// pcommon cannot handle the resulting *agentconfig.C values.
			// "conf_slice": []*agentconfig.C{agentconfig.MustNewConfigFrom(mapstr.M{"k": "v"})},

			// TODO: concretely-typed maps — ConvertNonPrimitive only handles
			// map[string]any and mapstr.M; all other map types fall to
			// "unknown type: <T>". Beats go-structform handles each via its own fold.
			// "map_str":    map[string]string{"key": "value"},
			// "map_mapstr": map[string]mapstr.M{"nested": {"k": "v"}},
			// "map_f64":    map[string]float64{"pi": 3.14},
			// "map_bool":   map[string]bool{"flag": true},
			// "map_u64":    map[string]uint64{"n": 1},
			// "map_int":    map[string]int{"a": 1},
			// "map_struct": map[string]struct{}{},
			// "map_byte":   map[string]byte{"b": 'A'},

			// TODO: pointer types — ConvertNonPrimitive has no pointer-unwrapping;
			// *time.Time and *mapstr.M produce "unknown type: *<T>". Beats
			// go-structform dereferences pointers and serialises normally.
			// "time_ptr":   &fixedTime,
			// "mapstr_ptr": &mapstr.M{"key": "val"},

			// TODO: domain-specific struct-pointer slices — ConvertNonPrimitive's
			// generic slice path stores each pointer element as interface{} in a
			// []any; pcommon.Value.FromRaw cannot handle arbitrary struct pointers
			// and logs "<Invalid value type *T>" for each element, storing null.
			// Beats go-structform serialises each struct via JSON marshal/unmarshal
			// to a full JSON object.
			// Verified failures:
			//   field "x509_certs"     — Beats: [{...full cert fields...}], OTel: [null]
			//   field "beat_info_slice" — Beats: [{...full Info fields...}], OTel: [null]
			// "x509_certs":      []*x509.Certificate{{}},
			// "beat_info_slice": []*beat.Info{{Name: "example"}},
		},
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// ── Beats Elasticsearch output path ──────────────────────────────────────
	// Use outputs.Load to build the Beats ES output, which internally calls
	// elasticsearch.NewEventEncoderFactory.  The factory is exposed via
	// group.EncoderFactory so we can pre-encode the event exactly as the real
	// pipeline does, then publish through the ES client to capture the raw JSON.
	beatsDocCh := make(chan []byte, 1)
	beatsMockES := newMockES(t, func(_ mockesapi.Action, event []byte) int {
		beatsDocCh <- event
		return http.StatusOK
	})
	beatsSrv := httptest.NewServer(beatsMockES)
	t.Cleanup(beatsSrv.Close)

	beatsGroup, err := outputs.Load(
		testIndexManager{},
		beat.Info{Name: "testbeat", Version: "0.0.0", Logger: logger},
		nil,
		"elasticsearch",
		agentconfig.MustNewConfigFrom(mapstr.M{"hosts": []string{beatsSrv.URL}}),
	)
	require.NoError(t, err)

	// Pre-encode the event using the factory (identical to what the pipeline does).
	beatsBatch := outest.NewBatch(beatEvent)
	beatsEnc := beatsGroup.EncoderFactory()
	require.Len(t, beatsBatch.Events(), 1)
	beatsBatch.Events()[0], _ = beatsEnc.EncodeEntry(beatsBatch.Events()[0])
	require.Len(t, beatsGroup.Clients, 1)
	beatsClient, ok := beatsGroup.Clients[0].(outputs.NetworkClient)
	require.True(t, ok, "ES output client must implement outputs.NetworkClient")
	require.NoError(t, beatsClient.Connect(ctx))
	require.NoError(t, beatsClient.Publish(ctx, beatsBatch))

	var beatsDoc []byte
	select {
	case beatsDoc = <-beatsDocCh:
	case <-ctx.Done():
		t.Fatal("timed out waiting for Beats ES output to deliver document to mock server")
	}

	// ── OTel path: otelconsumer → OTel ES exporter (bodymap) ─────────────────
	otelDocCh := make(chan []byte, 1)
	otelMockES := newMockES(t, func(_ mockesapi.Action, event []byte) int {
		otelDocCh <- event
		return http.StatusOK
	})
	otelSrv := httptest.NewServer(otelMockES)

	f := elasticsearchexporter.NewFactory()
	cfg, ok := f.CreateDefaultConfig().(*elasticsearchexporter.Config)
	require.Truef(t, ok, "elasticsearchexporter config must be of type *elasticsearchexporter.Config")
	cfg.Endpoints = []string{otelSrv.URL}
	// Reduce the batch flush timeout so the test does not wait the default 10s.
	qb := cfg.QueueBatchConfig.Get()
	qb.NumConsumers = 1
	qb.Batch.Get().FlushTimeout = 50 * time.Millisecond

	esExp, err := f.CreateLogs(ctx, exportertest.NewNopSettings(f.Type()), cfg)
	require.NoError(t, err)
	require.NoError(t, esExp.Start(ctx, componenttest.NewNopHost()))

	logConsumer, err := consumer.NewLogs(func(ctx context.Context, ld plog.Logs) error {
		return esExp.ConsumeLogs(ctx, ld)
	})
	require.NoError(t, err)

	oc := &otelConsumer{
		observer:     outputs.NewNilObserver(),
		logsConsumer: logConsumer,
		beatInfo:     beat.Info{Name: "testbeat", Version: "0.0.0"},
		log:          logger.Named("otelconsumer"),
		retry:        retryConfig{init: 1 * time.Millisecond, max: 2 * time.Millisecond},
	}

	otelBatch := outest.NewBatch(beatEvent)
	require.NoError(t, oc.Publish(ctx, otelBatch))

	var otelDoc []byte
	select {
	case otelDoc = <-otelDocCh:
	case <-ctx.Done():
		t.Fatal("timed out waiting for OTel exporter to deliver document to mock server")
	}

	// ── Comparison ────────────────────────────────────────────────────────────
	// assert.JSONEq normalises numbers (so "2" == "2.0"), hiding float comparison bugs.
	// Compare raw JSON tokens directly so that integer-vs-float differences are visible.
	beats := rawJSONFields(t, beatsDoc)
	otel := rawJSONFields(t, otelDoc)
	assert.Lenf(t, beats, len(otel), "top-level field count differs: beats=%d otel=%d", len(beats), len(otel))
	for field := range otel {
		assert.Containsf(t, beats, field, "unexpected field %q in OTel document", field)
	}

	// Fields whose values are JSON objects: Go map iteration order is non-deterministic
	// so the two serialisers may produce different key orderings. Compare each nested
	// field individually rather than comparing the raw token.
	nestedObjectFields := map[string]bool{
		"mapstr_nested": true,
		"map_any":       true,
	}

	// Compare all scalar and slice fields as raw JSON tokens.
	for field, beatsRaw := range beats {
		if field == "mapstr_slice" {
			// Elements contain only integers and strings so JSONEq does not hide
			// any float divergence while tolerating key-ordering differences.
			assert.JSONEqf(t, string(beats[field]), string(otel[field]), "mapstr_slice should be serialized identically")
			continue
		}

		if nestedObjectFields[field] {
			require.Contains(t, otel, field, "missing field %q in otel output", field)
			beatsNested := rawJSONFields(t, beats[field])
			otelNested := rawJSONFields(t, otel[field])
			assert.Lenf(t, beatsNested, len(otelNested), "%s field count differs", field)
			for nestedField, beatsNestedRaw := range beatsNested {
				otelNestedRaw, ok := otelNested[nestedField]
				assert.True(t, ok, "%s.%s missing from OTel document", field, nestedField)
				assert.Equal(t, string(beatsNestedRaw), string(otelNestedRaw), "%s.%s should be serialized identically", field, nestedField)
			}
			continue
		}

		otelRaw, ok := otel[field]
		if !assert.True(t, ok, "field %q missing from OTel document", field) {
			continue
		}
		assert.Equal(t, string(beatsRaw), string(otelRaw), "field %q: Beats=%s OTel=%s", field, beatsRaw, otelRaw)
	}
}

// rawJSONFields parses a JSON object and returns a map of field name to raw
// JSON token, preserving the exact byte form of each value so that "2" and
// "2.0" remain distinguishable (unlike a full json.Unmarshal which converts
// both to float64(2)).
func rawJSONFields(t *testing.T, data []byte) map[string]json.RawMessage {
	t.Helper()
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &m), "failed to parse JSON document: %s", data)
	return m
}

// newMockES creates a mock-es APIHandler that calls handler for each document
// in every bulk request.  The handler receives the parsed action and the raw
// document JSON bytes, and returns the HTTP status to report for that action.
func newMockES(t *testing.T, handler func(mockesapi.Action, []byte) int) *mockesapi.APIHandler {
	t.Helper()
	return mockesapi.NewDeterministicAPIHandler(
		uuid.Must(uuid.NewV4()),
		"",  // clusterUUID — empty is fine for tests
		nil, // meterProvider — nil uses the global no-op provider
		time.Now().Add(time.Hour),
		0,  // no artificial delay
		10, // history cap
		handler,
	)
}

// testIndexManager is a minimal outputs.IndexManager that always selects a
// fixed index name.  It is used when constructing the Beats ES output via
// outputs.Load in tests that do not need real index management.
type testIndexManager struct{}

func (testIndexManager) BuildSelector(_ *agentconfig.C) (outputs.IndexSelector, error) {
	return testIndexSelector{}, nil
}

type testIndexSelector struct{}

func (testIndexSelector) Select(_ *beat.Event) (string, error) {
	return "test-index", nil
}
