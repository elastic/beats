// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"errors"
	"strings"
	"time"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/otel/otelctx"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// parseEvent converts a log record into a beat.Event.
func parseEvent(logRecord *plog.LogRecord, index string) (beat.Event, error) {
	fields, ok := parseEventFields(logRecord)
	if !ok {
		return beat.Event{}, consumererror.NewPermanent(errors.New("invalid beats event body, expected a map, got: " + logRecord.Body().Type().String()))
	}

	timestamp, ok := parseEventTimestamp(fields)
	if ok {
		// written again by the json codec when the event is serialized
		delete(fields, beat.TimestampFieldKey)
	} else {
		timestamp = logRecord.ObservedTimestamp().AsTime()
	}

	return beat.Event{
		Timestamp: timestamp,
		Meta:      extractMetadataFields(fields, index),
		Fields:    fields,
	}, nil
}

func parseEventFields(logRecord *plog.LogRecord) (map[string]any, bool) {
	if logRecord.Body().Type() != pcommon.ValueTypeMap {
		return nil, false
	}
	return logRecord.Body().Map().AsRaw(), true
}

func parseEventTimestamp(logRecordBody map[string]any) (time.Time, bool) {
	timestamp, ok := logRecordBody[beat.TimestampFieldKey]
	if !ok {
		return time.Time{}, false
	}
	if typedVal, ok := timestamp.(string); ok {
		t, err := time.Parse("2006-01-02T15:04:05.000Z", typedVal)
		if err != nil {
			return time.Time{}, false
		}
		return t, true
	}
	return time.Time{}, false
}

// extractMetadataFields removes `@metadata` (present when the receiver runs with include_metadata)
// from the event fields and returns it as the event metadata.
// A non-empty index is the `index` explicitly configured on the exporter and takes
// precedence over the beat name set by the receiver, matching the Logstash output.
func extractMetadataFields(fields map[string]any, index string) mapstr.M {
	meta, ok := fields[beat.MetadataFieldKey].(map[string]any)
	if !ok {
		return nil
	}
	delete(fields, beat.MetadataFieldKey)

	// we are deleting the @metadata.version and @metadata.type keys because they are already present in the event metadata
	// See https://github.com/elastic/beats/blob/v9.3.3/libbeat/outputs/codec/json/event.go#L43-L54
	delete(meta, otelctx.MetadataVersionKey)
	delete(meta, "type")

	if index != "" {
		meta["beat"] = strings.ToLower(index)
	}
	return meta
}
