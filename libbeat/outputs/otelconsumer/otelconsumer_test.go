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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestPublish(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	event1 := beat.Event{Fields: mapstr.M{"field": 1}}
	event2 := beat.Event{Fields: mapstr.M{"field": 2}}
	event3 := beat.Event{Fields: mapstr.M{"field": 3}}
	event4 := beat.Event{Meta: mapstr.M{"_id": "abc123"}}

	makeOtelConsumer := func(t *testing.T, consumeFn func(ctx context.Context, ld plog.Logs) error) *otelConsumer {
		t.Helper()

		logger := logptest.NewTestingLogger(t, "")
		logConsumer, err := consumer.NewLogs(consumeFn)
		assert.NoError(t, err)
		consumer := &otelConsumer{
			observer:     outputs.NewNilObserver(),
			logsConsumer: logConsumer,
			beatInfo:     beat.Info{},
			log:          logger.Named("otelconsumer"),
		}
		return consumer
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

		var subFields = []string{"dataset", "namespace", "type"}
		for _, subField := range subFields {
			gotValue, ok := attributes.Get("data_stream." + subField)
			require.True(t, ok, fmt.Sprintf("data_stream.%s not found on log record attribute", subField))
			assert.EqualValues(t, dataStreamField[subField], gotValue.AsRaw())
		}

	})

	t.Run("retries the batch on non-permanent consumer error", func(t *testing.T) {
		batch := outest.NewBatch(event1, event2, event3)

		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			return errors.New("consume error")
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.Error(t, err)
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
		assert.Error(t, err)
		assert.True(t, consumererror.IsPermanent(err))
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchDrop, batch.Signals[0].Tag)
	})

	t.Run("retries on context cancelled", func(t *testing.T) {
		batch := outest.NewBatch(event1, event2, event3)

		otelConsumer := makeOtelConsumer(t, func(ctx context.Context, ld plog.Logs) error {
			return context.Canceled
		})

		err := otelConsumer.Publish(ctx, batch)
		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Len(t, batch.Signals, 1)
		assert.Equal(t, outest.BatchRetry, batch.Signals[0].Tag)
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
}
