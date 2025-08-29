// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"context"
	"errors"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

func parseEvent(ctx context.Context, logRecord *plog.LogRecord) (beat.Event, error) {
	metadata := parseEventMetadata(ctx)
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

func parseEventMetadata(ctx context.Context) map[string]any {
	ctxData := client.FromContext(ctx)
	var beatName, beatVersion string
	if v := ctxData.Metadata.Get("beat_name"); len(v) > 0 {
		beatName = v[0]
	}
	if v := ctxData.Metadata.Get("beat_version"); len(v) > 0 {
		beatVersion = v[0]
	}
	return map[string]any{
		"beat":    beatName,
		"version": beatVersion,
	}
}

func isBeatsEvent(metadata map[string]any) bool {
	v, ok := metadata["beat"]
	return ok && v != nil && v != ""
}

// GetBeatVersion retrieves the version of the beat from the context metadata.
// If the version is not found, it returns an empty string.
func GetBeatVersion(ctx context.Context) string {
	if version, ok := parseEventMetadata(ctx)["version"]; ok {
		return version.(string)
	}
	return ""
}
