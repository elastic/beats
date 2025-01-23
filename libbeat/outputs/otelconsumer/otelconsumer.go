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
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

const (
	// esDocumentIDAttribute is the attribute key used to store the document ID in the log record.
	esDocumentIDAttribute = "elasticsearch.document_id"
)

func init() {
	outputs.RegisterType("otelconsumer", makeOtelConsumer)
}

type otelConsumer struct {
	observer     outputs.Observer
	logsConsumer consumer.Logs
	beatInfo     beat.Info
	log          *logp.Logger
}

func makeOtelConsumer(_ outputs.IndexManager, beat beat.Info, observer outputs.Observer, cfg *config.C) (outputs.Group, error) {

	out := &otelConsumer{
		observer:     observer,
		logsConsumer: beat.LogConsumer,
		beatInfo:     beat,
		log:          logp.NewLogger("otelconsumer"),
	}

	ocConfig := defaultConfig()
	if err := cfg.Unpack(&ocConfig); err != nil {
		return outputs.Fail(err)
	}
	return outputs.Success(ocConfig.Queue, -1, 0, nil, out)
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
	pLogs := plog.NewLogs()
	resourceLogs := pLogs.ResourceLogs().AppendEmpty()
	sourceLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecords := sourceLogs.LogRecords()

	events := batch.Events()
	for _, event := range events {
		logRecord := logRecords.AppendEmpty()
		meta := event.Content.Meta.Clone()
		meta["beat"] = out.beatInfo.Beat
		meta["version"] = out.beatInfo.Version
		meta["type"] = "_doc"

		if id, ok := meta["_id"]; ok {
			// Set a log record attribute for the document ID, this is used for deduplication
			// in the elasticsearchexporter.
			//
			// See https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/36882.
			switch id := id.(type) {
			case string:
				logRecord.Attributes().PutStr(esDocumentIDAttribute, id)
			}
		}

		beatEvent := event.Content.Fields.Clone()
		beatEvent["@timestamp"] = event.Content.Timestamp
		beatEvent["@metadata"] = meta
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(event.Content.Timestamp))
		pcommonEvent := mapstrToPcommonMap(beatEvent)
		pcommonEvent.CopyTo(logRecord.Body().SetEmptyMap())
	}

	err := out.logsConsumer.ConsumeLogs(ctx, pLogs)
	if err != nil {
		// Permanent errors shouldn't be retried. This tipically means
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
			batch.Retry()
		}

		return fmt.Errorf("failed to send batch events to otel collector: %w", err)
	}

	batch.ACK()
	st.NewBatch(len(events))
	st.AckedEvents(len(events))
	return nil
}

func (out *otelConsumer) String() string {
	return "otelconsumer"
}

// mapstrToPcommonMap is necessary to convert from Beats mapstr to
// Otel Map.  This step could be avoided if we choose to encode the
// Body as a slice of bytes.
func mapstrToPcommonMap(m mapstr.M) pcommon.Map {
	out := pcommon.NewMap()
	for k, v := range m {
		switch x := v.(type) {
		case string:
			out.PutStr(k, x)
		case []string:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]string) {
				newVal := dest.AppendEmpty()
				newVal.SetStr(i)
			}
		case int:
			out.PutInt(k, int64(v.(int)))
		case []int:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]int) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(int64(i))
			}
		case int8:
			out.PutInt(k, int64(v.(int8)))
		case []int8:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]int8) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(int64(i))
			}
		case int16:
			out.PutInt(k, int64(v.(int16)))
		case []int16:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]int16) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(int64(i))
			}
		case int32:
			out.PutInt(k, int64(v.(int32)))
		case []int32:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]int32) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(int64(i))
			}
		case int64:
			out.PutInt(k, v.(int64))
		case []int64:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]int64) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(i)
			}
		case uint:
			out.PutInt(k, int64(v.(uint)))
		case []uint:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]uint) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(int64(i))
			}
		case uint8:
			out.PutInt(k, int64(v.(uint8)))
		case []uint8:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]uint8) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(int64(i))
			}
		case uint16:
			out.PutInt(k, int64(v.(uint16)))
		case []uint16:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]uint16) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(int64(i))
			}
		case uint32:
			out.PutInt(k, int64(v.(uint32)))
		case []uint32:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]uint32) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(int64(i))
			}
		case uint64:
			out.PutInt(k, int64(v.(uint64)))
		case []uint64:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]uint64) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(int64(i))
			}
		case float32:
			out.PutDouble(k, float64(v.(float32)))
		case []float32:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]float32) {
				newVal := dest.AppendEmpty()
				newVal.SetDouble(float64(i))
			}
		case float64:
			out.PutDouble(k, v.(float64))
		case []float64:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]float64) {
				newVal := dest.AppendEmpty()
				newVal.SetDouble(i)
			}
		case bool:
			out.PutBool(k, x)
		case []bool:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]bool) {
				newVal := dest.AppendEmpty()
				newVal.SetBool(i)
			}
		case mapstr.M:
			dest := out.PutEmptyMap(k)
			newMap := mapstrToPcommonMap(x)
			newMap.CopyTo(dest)
		case []mapstr.M:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]mapstr.M) {
				newVal := dest.AppendEmpty()
				newMap := mapstrToPcommonMap(i)
				newMap.CopyTo(newVal.SetEmptyMap())
			}
		case time.Time:
			out.PutInt(k, x.UnixMilli())
		case []time.Time:
			dest := out.PutEmptySlice(k)
			for _, i := range v.([]time.Time) {
				newVal := dest.AppendEmpty()
				newVal.SetInt(i.UnixMilli())
			}
		default:
			out.PutStr(k, fmt.Sprintf("unknown type: %T", x))
		}
	}
	return out
}
