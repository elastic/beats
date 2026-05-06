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
	"encoding"
	"errors"
	"fmt"
	"math"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/libbeat/otel/otelctx"
	"github.com/elastic/beats/v7/libbeat/otel/otelmap"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

const (
	// esDocumentIDAttribute is the attribute key used to store the document ID in the log record.
	esDocumentIDAttribute = "elasticsearch.document_id"
	otelTimestampLayout   = "2006-01-02T15:04:05.000Z"

	retryBackoffInit = 1 * time.Second
	retryBackoffMax  = 60 * time.Second
)

var errDirectEncodeUnsupported = errors.New("unsupported direct-encoding value")

type signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

type unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

type floating interface {
	~float32 | ~float64
}

type mapstrOrMap interface {
	mapstr.M | map[string]any
}

type retryConfig struct {
	init time.Duration
	max  time.Duration
}

type otelConsumer struct {
	observer       outputs.Observer
	logsConsumer   consumer.Logs
	beatInfo       beat.Info
	log            *logp.Logger
	isReceiverTest bool // whether we are running in receivertest context

	retry        retryConfig
	retryBackoff backoff.Backoff
	backoffInit  sync.Once
}

func MakeOtelConsumer(beat beat.Info, observer outputs.Observer) (outputs.Group, error) {
	isReceiverTest := os.Getenv("OTELCONSUMER_RECEIVERTEST") == "1"

	retry := retryConfig{init: retryBackoffInit, max: retryBackoffMax}
	if isReceiverTest {
		retry = retryConfig{init: 1 * time.Millisecond, max: 2 * time.Millisecond}
	}

	// Default to runtime.NumCPU() workers
	clients := make([]outputs.Client, 0, runtime.NumCPU())
	for range runtime.NumCPU() {
		clients = append(clients, &otelConsumer{
			observer:       observer,
			logsConsumer:   beat.LogConsumer,
			beatInfo:       beat,
			log:            beat.Logger.Named("otelconsumer"),
			isReceiverTest: isReceiverTest,
			retry:          retry,
		})
	}

	return outputs.Group{Clients: clients}, nil
}

// Close is a noop for otelconsumer
func (out *otelConsumer) Close() error {
	return nil
}

// Publish converts Beat events to Otel format and sends them to the Otel collector
func (out *otelConsumer) Publish(ctx context.Context, batch publisher.Batch) error {
	switch {
	case out.logsConsumer != nil:
		return out.logsPublish(ctx, batch)
	default:
		panic(fmt.Errorf("an otel consumer must be specified"))
	}
}

func (out *otelConsumer) logsPublish(ctx context.Context, batch publisher.Batch) error {
	st := out.observer
	events := batch.Events()
	st.NewBatch(len(events))

	pLogs := plog.NewLogs()
	resourceLogs := pLogs.ResourceLogs().AppendEmpty()
	sourceLogs := resourceLogs.ScopeLogs().AppendEmpty()

	// add bodymap mapping mode on scope attributes
	sourceLogs.Scope().Attributes().PutStr("elastic.mapping.mode", "bodymap")

	logRecords := sourceLogs.LogRecords()

	// Convert the batch of events to Otel plog.Logs. The encoding we
	// choose here is to set all fields in a Map in the Body of the log
	// record. Each log record encodes a single beats event.
	// This way we have full control over the final structure of the log in the
	// destination, as long as the exporter allows it.
	// For example, the elasticsearchexporter has an encoding specifically for this.
	// See https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/35444.
	logRecords.EnsureCapacity(len(events))
	for _, event := range events {
		logRecord := logRecords.AppendEmpty()
		if err := fillLogRecordFromEvent(logRecord, event, out.beatInfo, out.log, out.isReceiverTest); err != nil {
			out.log.Errorf("received an error while converting map to plog.Log, some fields might be missing: %v", err)
		}
	}

	out.backoffInit.Do(func() {
		out.retryBackoff = backoff.NewEqualJitterBackoff(ctx.Done(), out.retry.init, out.retry.max)
	})

	err := out.logsConsumer.ConsumeLogs(otelctx.NewConsumerContext(ctx, out.beatInfo), pLogs)
	if err != nil {
		// Queue full errors are expected backpressure signals, not true errors.
		// Skip logging to avoid log spam since we already track this via metrics.
		if !errors.Is(err, exporterhelper.ErrQueueIsFull) {
			out.log.Errorf("failed to publish batch events to otel collector pipeline: %v", err)
		}

		// Permanent errors shouldn't be retried. This typically means
		// the data cannot be serialized by the exporter that is attached
		// to the pipeline or when the destination refuses the data because
		// it cannot decode it. Retrying in this case is useless.
		//
		// See https://github.com/open-telemetry/opentelemetry-collector/blob/1c47d89/receiver/doc.go#L23-L40
		if consumererror.IsPermanent(err) {
			st.PermanentErrors(len(events))
			batch.Drop()
		} else {
			st.RetryableErrors(len(events))
			if !out.retryBackoff.Wait() {
				batch.Cancelled()
				return nil
			}
			batch.Retry()
		}
		return nil
	}

	batch.ACK()
	st.AckedEvents(len(events))
	out.retryBackoff.Reset()
	return nil
}

func (out *otelConsumer) String() string {
	return "otelconsumer"
}

func fillLogRecordFromEvent(logRecord plog.LogRecord, event publisher.Event, beatInfo beat.Info, log *logp.Logger, isReceiverTest bool) error {
	beatEvent := prepareLogRecordFromEvent(logRecord, event, log, isReceiverTest)
	metadata := logBodyMetadata(event, beatInfo)

	// Fast-path the common Beat value shapes directly into pdata. If we hit a
	// less common type, fall back to the legacy clone + ConvertNonPrimitive +
	// FromRaw path so behavior stays identical.
	err := tryEncodeLogRecordBodyDirect(logRecord, beatEvent, event.Content.Timestamp, metadata)
	if err == nil || !errors.Is(err, errDirectEncodeUnsupported) {
		return err
	}
	return encodeLogRecordBodyFromRawWithTimestamp(logRecord, beatEvent, event.Content.Timestamp, metadata)
}

func prepareLogRecordFromEvent(logRecord plog.LogRecord, event publisher.Event, log *logp.Logger, isReceiverTest bool) mapstr.M {
	if id, ok := event.Content.Meta["_id"]; ok {
		// Specify the id as an attribute used by the elasticsearchexporter
		// to set the final document ID in Elasticsearch.
		// When using the bodymap encoding in the exporter all attributes
		// are stripped out of the final Elasticsearch document.
		//
		// See https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/36882.
		switch id := id.(type) {
		case string:
			logRecord.Attributes().PutStr(esDocumentIDAttribute, id)

			// The receivertest package needs a unique attribute to track generated ids.
			// When receivertest allows this to be customized we can remove this condition.
			// See https://github.com/open-telemetry/opentelemetry-collector/issues/12003.
			if isReceiverTest {
				logRecord.Attributes().PutStr(receivertest.UniqueIDAttrName, id)
			}
		}
	}

	// if pipeline field is set on event metadata
	if pipeline, err := event.Content.Meta.GetValue("pipeline"); err == nil {
		if s, ok := pipeline.(string); ok {
			logRecord.Attributes().PutStr("elasticsearch.ingest_pipeline", s)
		}
	}

	beatEvent := event.Content.Fields
	if beatEvent == nil {
		beatEvent = mapstr.M{}
	}
	logRecord.SetTimestamp(pcommon.NewTimestampFromTime(event.Content.Timestamp))

	// Set the timestamp for when the event was first seen by the pipeline.
	observedTimestamp := logRecord.Timestamp()
	if created, err := beatEvent.GetValue("event.created"); err == nil {
		switch created := created.(type) {
		case time.Time:
			observedTimestamp = pcommon.NewTimestampFromTime(created)
		case common.Time:
			observedTimestamp = pcommon.NewTimestampFromTime(time.Time(created))
		default:
			log.Warnf("Invalid 'event.created' type (%T); using log timestamp as observed timestamp.", created)
		}
	}
	logRecord.SetObservedTimestamp(observedTimestamp)

	// if data_stream field is set on beatEvent. Add it to logrecord.Attributes to support dynamic indexing
	if val, _ := beatEvent.GetValue("data_stream"); val != nil {
		// If the below sub fields do not exist, it will return empty string.
		subFields := []string{"dataset", "namespace", "type"}

		for _, subField := range subFields {
			value, err := beatEvent.GetValue("data_stream." + subField)
			if vStr, ok := value.(string); ok && err == nil {
				// set log record attribute only if value is non empty
				logRecord.Attributes().PutStr("data_stream."+subField, vStr)
			}
		}
	}

	return beatEvent
}

func logBodyMetadata(event publisher.Event, beatInfo beat.Info) mapstr.M {
	if !beatInfo.IncludeMetadata {
		return nil
	}

	meta := event.Content.Meta.Clone()
	if meta == nil {
		meta = mapstr.M{}
	}
	meta["beat"] = beatInfo.Beat
	meta["version"] = beatInfo.Version
	meta["type"] = "_doc"
	return meta
}

func encodeLogRecordBodyFromRawWithTimestamp(logRecord plog.LogRecord, beatEvent mapstr.M, timestamp time.Time, metadata mapstr.M) error {
	beatEvent = beatEvent.Clone()
	if beatEvent == nil {
		beatEvent = mapstr.M{}
	}
	beatEvent["@timestamp"] = timestamp
	if metadata != nil {
		beatEvent["@metadata"] = metadata
	}
	otelmap.ConvertNonPrimitive(beatEvent)
	return logRecord.Body().SetEmptyMap().FromRaw(map[string]any(beatEvent))
}

func tryEncodeLogRecordBodyDirect(logRecord plog.LogRecord, beatEvent mapstr.M, timestamp time.Time, metadata mapstr.M) error {
	bodyMap := logRecord.Body().SetEmptyMap()
	capacity := len(beatEvent) + 1
	if metadata != nil {
		capacity++
	}
	bodyMap.EnsureCapacity(capacity)
	bodyMap.PutStr("@timestamp", otelTimestamp(timestamp))
	if metadata != nil {
		if err := encodeValueDirect(bodyMap.PutEmpty("@metadata"), metadata); err != nil {
			return err
		}
	}
	for key, value := range beatEvent {
		if key == "@timestamp" || (metadata != nil && key == "@metadata") {
			continue
		}
		if err := encodeValueDirect(bodyMap.PutEmpty(key), value); err != nil {
			return err
		}
	}
	return nil
}

func encodeMapDirect[T mapstrOrMap](dest pcommon.Map, src T) error {
	dest.EnsureCapacity(len(src))
	for key, value := range src {
		if err := encodeValueDirect(dest.PutEmpty(key), value); err != nil {
			return err
		}
	}
	return nil
}

func encodeMapSliceDirect[T mapstrOrMap](dest pcommon.Slice, src []T) error {
	dest.EnsureCapacity(len(src))
	for _, item := range src {
		if err := encodeMapDirect(dest.AppendEmpty().SetEmptyMap(), item); err != nil {
			return err
		}
	}
	return nil
}

func encodeAnySliceDirect(dest pcommon.Slice, src []any) error {
	dest.EnsureCapacity(len(src))
	for _, item := range src {
		if err := encodeValueDirect(dest.AppendEmpty(), item); err != nil {
			return err
		}
	}
	return nil
}

func encodeTimeSliceDirect(dest pcommon.Slice, src []time.Time) error {
	dest.EnsureCapacity(len(src))
	for _, item := range src {
		dest.AppendEmpty().SetStr(otelTimestamp(item))
	}
	return nil
}

func encodeCommonTimeSliceDirect(dest pcommon.Slice, src []common.Time) error {
	dest.EnsureCapacity(len(src))
	for _, item := range src {
		dest.AppendEmpty().SetStr(otelTimestamp(time.Time(item)))
	}
	return nil
}

func encodeStringSliceDirect(dest pcommon.Slice, src []string) error {
	dest.EnsureCapacity(len(src))
	for _, item := range src {
		dest.AppendEmpty().SetStr(item)
	}
	return nil
}

func encodeBoolSliceDirect(dest pcommon.Slice, src []bool) error {
	dest.EnsureCapacity(len(src))
	for _, item := range src {
		dest.AppendEmpty().SetBool(item)
	}
	return nil
}

func encodeFloatSliceDirect[T floating](dest pcommon.Slice, src []T) error {
	dest.EnsureCapacity(len(src))
	for _, item := range src {
		dest.AppendEmpty().SetDouble(float64(item))
	}
	return nil
}

func encodeSignedSliceDirect[T signed](dest pcommon.Slice, src []T) error {
	dest.EnsureCapacity(len(src))
	for _, item := range src {
		dest.AppendEmpty().SetInt(int64(item))
	}
	return nil
}

func encodeUnsignedSliceDirect[T unsigned](dest pcommon.Slice, src []T) error {
	dest.EnsureCapacity(len(src))
	for _, item := range src {
		dest.AppendEmpty().SetInt(maskUnsignedInt(uint64(item)))
	}
	return nil
}

func maskUnsignedInt(value uint64) int64 {
	return int64(value & uint64(math.MaxInt64)) //nolint:gosec // mask clears bit 63, conversion is safe
}

func encodeValueDirect(dest pcommon.Value, value any) error {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		dest.SetStr(v)
		return nil
	case int:
		dest.SetInt(int64(v))
		return nil
	case int8:
		dest.SetInt(int64(v))
		return nil
	case int16:
		dest.SetInt(int64(v))
		return nil
	case int32:
		dest.SetInt(int64(v))
		return nil
	case int64:
		dest.SetInt(v)
		return nil
	case uint:
		dest.SetInt(maskUnsignedInt(uint64(v)))
		return nil
	case uint8:
		dest.SetInt(int64(v))
		return nil
	case uint16:
		dest.SetInt(int64(v))
		return nil
	case uint32:
		dest.SetInt(int64(v))
		return nil
	case uint64:
		dest.SetInt(maskUnsignedInt(v))
		return nil
	case float32:
		dest.SetDouble(float64(v))
		return nil
	case float64:
		dest.SetDouble(v)
		return nil
	case bool:
		dest.SetBool(v)
		return nil
	case mapstr.M:
		return encodeMapDirect(dest.SetEmptyMap(), v)
	case map[string]any:
		return encodeMapDirect(dest.SetEmptyMap(), v)
	case []mapstr.M:
		return encodeMapSliceDirect(dest.SetEmptySlice(), v)
	case []map[string]any:
		return encodeMapSliceDirect(dest.SetEmptySlice(), v)
	case []any:
		return encodeAnySliceDirect(dest.SetEmptySlice(), v)
	case time.Time:
		dest.SetStr(otelTimestamp(v))
		return nil
	case common.Time:
		dest.SetStr(otelTimestamp(time.Time(v)))
		return nil
	case time.Duration:
		dest.SetInt(int64(v))
		return nil
	case encoding.TextMarshaler:
		text, err := v.MarshalText()
		if err != nil {
			dest.SetStr(fmt.Sprintf("error converting %T to string: %s", v, err))
			return nil
		}
		dest.SetStr(string(text))
		return nil
	case []time.Time:
		return encodeTimeSliceDirect(dest.SetEmptySlice(), v)
	case []common.Time:
		return encodeCommonTimeSliceDirect(dest.SetEmptySlice(), v)
	case []string:
		return encodeStringSliceDirect(dest.SetEmptySlice(), v)
	case []bool:
		return encodeBoolSliceDirect(dest.SetEmptySlice(), v)
	case []float32:
		return encodeFloatSliceDirect(dest.SetEmptySlice(), v)
	case []float64:
		return encodeFloatSliceDirect(dest.SetEmptySlice(), v)
	case []int:
		return encodeSignedSliceDirect(dest.SetEmptySlice(), v)
	case []int8:
		return encodeSignedSliceDirect(dest.SetEmptySlice(), v)
	case []int16:
		return encodeSignedSliceDirect(dest.SetEmptySlice(), v)
	case []int32:
		return encodeSignedSliceDirect(dest.SetEmptySlice(), v)
	case []int64:
		return encodeSignedSliceDirect(dest.SetEmptySlice(), v)
	case []uint:
		return encodeUnsignedSliceDirect(dest.SetEmptySlice(), v)
	case []uint8:
		return encodeUnsignedSliceDirect(dest.SetEmptySlice(), v)
	case []uint16:
		return encodeUnsignedSliceDirect(dest.SetEmptySlice(), v)
	case []uint32:
		return encodeUnsignedSliceDirect(dest.SetEmptySlice(), v)
	case []uint64:
		return encodeUnsignedSliceDirect(dest.SetEmptySlice(), v)
	default:
		return errDirectEncodeUnsupported
	}
}

func otelTimestamp(t time.Time) string {
	return t.UTC().Format(otelTimestampLayout)
}
