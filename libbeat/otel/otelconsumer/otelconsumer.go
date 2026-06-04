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
)

const (
	// esDocumentIDAttribute is the attribute key used to store the document ID in the log record.
	esDocumentIDAttribute = "elasticsearch.document_id"

	// receivertestUniqueIDAttrName mirrors receivertest.UniqueIDAttrName.
	// It is duplicated here to avoid importing the receivertest package
	// (and pulling its testify/testing deps) into production binaries.
	receivertestUniqueIDAttrName = "test_id"

	retryBackoffInit = 1 * time.Second
	retryBackoffMax  = 60 * time.Second
)

// dataStreamAttributeKeys are the data_stream sub-fields promoted to log record
// attributes to support dynamic indexing. They double as both the mapstr lookup
// path and the attribute key, and are kept at package scope so they aren't
// rebuilt for every event.
var dataStreamAttributeKeys = [...]string{
	"data_stream.dataset",
	"data_stream.namespace",
	"data_stream.type",
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

	pLogs := out.eventsToLogs(events, &out.beatInfo)

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

// eventsToLogs converts a group of Beat events to a single plog.Logs, using the
// given beat.Info for metadata.
func (out *otelConsumer) eventsToLogs(events []publisher.Event, beatInfo *beat.Info) plog.Logs {
	pLogs := plog.NewLogs()
	resourceLogs := pLogs.ResourceLogs().AppendEmpty()
	sourceLogs := resourceLogs.ScopeLogs().AppendEmpty()

	// add bodymap mapping mode on scope attributes
	sourceLogs.Scope().Attributes().PutStr("elastic.mapping.mode", "bodymap")

	logRecords := sourceLogs.LogRecords()
	// Pre-size the record slice so it isn't repeatedly grown as we append one
	// record per event below.
	logRecords.EnsureCapacity(len(events))

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

	return pLogs
}

func (out *otelConsumer) String() string {
	return "otelconsumer"
}

func fillLogRecordFromEvent(logRecord plog.LogRecord, event publisher.Event, beatInfo beat.Info, log *logp.Logger, isReceiverTest bool) error {
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
				logRecord.Attributes().PutStr(receivertestUniqueIDAttrName, id)
			}
		}
	}

	// if pipeline field is set on event metadata
	if s, ok := event.Content.Meta["pipeline"].(string); ok {
		logRecord.Attributes().PutStr("elasticsearch.ingest_pipeline", s)
	}

	beatEvent := event.Content.Fields
	if beatEvent == nil {
		beatEvent = mapstr.M{}
	}
	logRecord.SetTimestamp(pcommon.NewTimestampFromTime(event.Content.Timestamp))

	// Set the timestamp for when the event was first seen by the pipeline.
	observedTimestamp := logRecord.Timestamp()
	if eventMap, ok := tryToMapStr(beatEvent["event"]); ok {
		switch created := eventMap["created"].(type) {
		case time.Time:
			observedTimestamp = pcommon.NewTimestampFromTime(created)
		case common.Time:
			observedTimestamp = pcommon.NewTimestampFromTime(time.Time(created))
		case nil:
			// not set
		default:
			log.Warnf("Invalid 'event.created' type (%T); using log timestamp as observed timestamp.", created)
		}
	}
	logRecord.SetObservedTimestamp(observedTimestamp)

	// if data_stream field is set on beatEvent. Add it to logrecord.Attributes to support dynamic indexing
	if ds, ok := tryToMapStr(beatEvent["data_stream"]); ok {
		for _, sub := range [...]string{"dataset", "namespace", "type"} {
			if vStr, ok := ds[sub].(string); ok {
				logRecord.Attributes().PutStr("data_stream."+sub, vStr)
			}
		}
	}

	bodyMap := logRecord.Body().SetEmptyMap()
	capacity := len(beatEvent) + 1 // +1 for @timestamp added below
	if beatInfo.IncludeMetadata {
		capacity++ // +1 for @metadata map added below
	}
	bodyMap.EnsureCapacity(capacity)
	if err := otelmap.FromMapstr(bodyMap, beatEvent); err != nil {
		return err
	}

	bodyMap.PutStr("@timestamp", otelmap.FormatTimestamp(event.Content.Timestamp))
	if beatInfo.IncludeMetadata {
		extra := [...]struct{ k, v string }{
			{"beat", beatInfo.Beat},
			{"version", beatInfo.Version},
			{"type", "_doc"},
		}
		pmeta := bodyMap.PutEmpty("@metadata").SetEmptyMap()
		pmeta.EnsureCapacity(len(event.Content.Meta) + len(extra))
		if err := otelmap.FromMapstr(pmeta, event.Content.Meta); err != nil {
			return err
		}
		for _, kv := range extra {
			pmeta.PutStr(kv.k, kv.v)
		}
	}
	return nil
}

func tryToMapStr(v interface{}) (mapstr.M, bool) {
	switch m := v.(type) {
	case mapstr.M:
		return m, true
	case map[string]interface{}:
		return mapstr.M(m), true
	default:
		return nil, false
	}
}
