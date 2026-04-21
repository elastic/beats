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
)

func parseEvent(logRecord *plog.LogRecord) (beat.Event, error) {
	fields, ok := parseEventFields(logRecord)
	if !ok {
		return beat.Event{}, consumererror.NewPermanent(errors.New("invalid beats event body, expected a map, got: " + logRecord.Body().Type().String()))
	}

	removeRedundantMetadataFields(fields)

	timestamp, ok := parseEventTimestamp(fields)
	if !ok {
		timestamp = logRecord.ObservedTimestamp().AsTime()
	}

	event := beat.Event{
		Timestamp: timestamp,
		Fields:    fields,
	}

	return event, nil
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

// removeRedundantMetadataFields removes certain metadata fields that will be generated again when the event is serialized.
// See https://github.com/elastic/beats/blob/v9.3.3/libbeat/outputs/codec/json/event.go#L43-L54
// Not removing these fields would create duplicate fields and bloat the final event size
func removeRedundantMetadataFields(fields map[string]any) {
	delete(fields, otelctx.MetadataBeatKey)
	delete(fields, otelctx.MetadataVersionKey)
	delete(fields, "type")
}
