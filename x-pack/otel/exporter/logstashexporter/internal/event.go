// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/libbeat/outputs/otelctx"
)

func parseEvent(ctx context.Context, logRecord *plog.LogRecord) (beat.Event, error) {
	metadata := otelctx.GetBeatEventMeta(ctx)
	if !isBeatsEvent(metadata) {
		return beat.Event{}, consumererror.NewPermanent(errors.New("invalid beats event metadata"))
	}

	fields, ok := parseEventFields(logRecord)
	if !ok {
		return beat.Event{}, consumererror.NewPermanent(errors.New("invalid beats event body, expected a map, got: " + logRecord.Body().Type().String()))
	}

	timestamp, ok := parseEventTimestamp(fields)
	if !ok {
		timestamp = logRecord.ObservedTimestamp().AsTime()
	}

	return beat.Event{
		Timestamp: timestamp,
		Meta:      metadata,
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

func isBeatsEvent(metadata map[string]any) bool {
	v, ok := metadata["beat"]
	return ok && v != nil && v != ""
}
