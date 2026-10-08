// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/otel/otelctx"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func parseEvent(logRecord *plog.LogRecord) (beat.Event, error) {
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
		Meta:      extractMetadataFields(fields),
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
// from the event fields and returns it as the event metadata, minus `beat`, `version` and `type`.
// The json codec writes `@metadata` with these keys itself when the event is serialized, so leaving
// them in the fields would duplicate them.
// See https://github.com/elastic/beats/blob/v9.3.3/libbeat/outputs/codec/json/event.go#L43-L54
// Top-level fields with these names are event data and are left alone.
func extractMetadataFields(fields map[string]any) mapstr.M {
	meta, ok := fields[beat.MetadataFieldKey].(map[string]any)
	if !ok {
		return nil
	}
	delete(fields, beat.MetadataFieldKey)
	delete(meta, otelctx.MetadataBeatKey)
	delete(meta, otelctx.MetadataVersionKey)
	delete(meta, "type")
	return meta
}
