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
	"github.com/elastic/elastic-agent-libs/mapstr"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

func init() {
	outputs.RegisterType("otelconsumer", makeOtelConsumer)
}

type otelConsumer struct {
	observer     outputs.Observer
	logsConsumer consumer.Logs
	beatInfo     beat.Info
}

func makeOtelConsumer(_ outputs.IndexManager, beat beat.Info, observer outputs.Observer, cfg *config.C) (outputs.Group, error) {

	out := &otelConsumer{
		observer:     observer,
		logsConsumer: beat.LogConsumer,
		beatInfo:     beat,
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

// Publish converts Beat events to Otel format and send to the next otel consumer
func (out *otelConsumer) Publish(ctx context.Context, batch publisher.Batch) error {
	switch {
	case out.logsConsumer != nil:
		return out.logsPublish(ctx, batch)
	default:
		panic(fmt.Errorf("an otel consumer must be specified"))
	}
}

func (out *otelConsumer) logsPublish(_ context.Context, batch publisher.Batch) error {
	defer batch.ACK()
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

		beatEvent := event.Content.Fields.Clone()
		beatEvent["@timestamp"] = event.Content.Timestamp
		beatEvent["@metadata"] = meta
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(event.Content.Timestamp))
		pcommonEvent := mapstrToPcommonMap(beatEvent)
		pcommonEvent.CopyTo(logRecord.Body().SetEmptyMap())
	}

	if err := out.logsConsumer.ConsumeLogs(context.TODO(), pLogs); err != nil {
		return fmt.Errorf("error otel log consumer: %w", err)
	}

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
		case int:
			out.PutInt(k, int64(v.(int)))
		case int8:
			out.PutInt(k, int64(v.(int8)))
		case int16:
			out.PutInt(k, int64(v.(int16)))
		case int32:
			out.PutInt(k, int64(v.(int32)))
		case int64:
			out.PutInt(k, v.(int64))
		case uint:
			out.PutInt(k, int64(v.(uint)))
		case uint8:
			out.PutInt(k, int64(v.(uint8)))
		case uint16:
			out.PutInt(k, int64(v.(uint16)))
		case uint32:
			out.PutInt(k, int64(v.(uint32)))
		case uint64:
			out.PutInt(k, int64(v.(uint64)))
		case float32:
			out.PutDouble(k, float64(v.(float32)))
		case float64:
			out.PutDouble(k, v.(float64))
		case bool:
			out.PutBool(k, x)
		case mapstr.M:
			dest := out.PutEmptyMap(k)
			newMap := mapstrToPcommonMap(x)
			newMap.CopyTo(dest)
		case time.Time:
			out.PutInt(k, x.UnixMilli())
		default:
			out.PutStr(k, fmt.Sprintf("unknown type: %T", x))
		}
	}
	return out
}
