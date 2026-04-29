// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"errors"
	"fmt"
	"time"
)

// batchCursorTimeLayout is the canonical layout used to persist progress_time
// and to render batch_start_time / batch_end_time for SOQL templates. It
// mirrors the Salesforce datetime literal grammar (RFC3339 with a numeric
// timezone offset and millisecond precision) so the value can be embedded
// directly in a SOQL WHERE clause.
const batchCursorTimeLayout = "2006-01-02T15:04:05.000Z07:00"

// supportedBatchTimeLayouts is the set of timestamp layouts that
// parseBatchCursorTime accepts when reading cursor values back from
// persisted state. It intentionally includes both the canonical batch
// layout above and the legacy Salesforce "Z0700" form (without the colon in
// the offset) that existing dateTimeCursor.FirstEventTime /
// dateTimeCursor.LastEventTime values are written in, so that an upgrade
// from the pre-batch module can resume without manual state migration.
var supportedBatchTimeLayouts = []string{
	batchCursorTimeLayout,
	"2006-01-02T15:04:05.000Z0700",
	time.RFC3339,
	time.RFC3339Nano,
	formatRFC3339Like,
}

// objectBatchWindow is a half-open (Start, End] time window used by the
// batched Object collection path to bound a single SOQL query. The end is
// clamped to the run's wall-clock end so the window never extends into the
// future.
type objectBatchWindow struct {
	Start time.Time
	End   time.Time
}

// formatBatchCursorTime renders t in the canonical batchCursorTimeLayout,
// normalized to UTC. This is the only formatter used when the input writes
// progress_time or exposes batch_start_time / batch_end_time to user
// templates, so every value that lands in persisted state or in a rendered
// SOQL query goes through it.
func formatBatchCursorTime(t time.Time) string {
	return t.UTC().Format(batchCursorTimeLayout)
}

// parseBatchCursorTime parses a Salesforce cursor timestamp persisted by any
// of the layouts in supportedBatchTimeLayouts. The result is always in UTC.
// An unparseable value is returned as an error rather than silently reset to
// zero so the input fails loud on corrupt state instead of quietly skipping
// or replaying data.
func parseBatchCursorTime(raw string) (time.Time, error) {
	for _, layout := range supportedBatchTimeLayouts {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported Salesforce cursor time format: %q", raw)
}

// nextObjectBatchWindow selects the next bounded catch-up window for Object
// collection, relative to runEnd (the wall-clock end of the current run).
//
// The window start is chosen from the persisted object cursor using the
// following priority:
//
//  1. progress_time - the batched-path watermark. Used on every run after
//     the first successful batched window. When first_event_time or
//     last_event_time were advanced past progress_time by an unbatched run
//     (i.e. the user toggled batch.enabled off and back on), the start is
//     projected forward to the latest of the three so the first re-enabled
//     window doesn't replay events the unbatched path already drained.
//  2. first_event_time - legacy real-time module watermark. Used the first
//     time the batched path runs on an install that was previously using
//     the unbatched real-time template (login / logout), so upgrades resume
//     from the last seen event instead of skipping or replaying data.
//  3. last_event_time - same idea as first_event_time, for templates that
//     only persisted last_event_time.
//  4. runEnd - batch.initial_interval - clean-install fallback when no
//     cursor is persisted.
//
// The window end is start + batch.window, clamped to runEnd. The returned
// bool is false when there is no more work to do in the current run
// (start >= runEnd), in which case runObjectBatches should stop iterating.
// An error is returned only when the persisted cursor value cannot be
// parsed; nextObjectBatchWindow intentionally surfaces that so an operator
// can see and fix the bad state rather than silently drop back to the
// initial-interval default.
func (s *salesforceInput) nextObjectBatchWindow(runEnd time.Time) (objectBatchWindow, bool, error) {
	batch, err := s.objectBatchConfig()
	if err != nil {
		return objectBatchWindow{}, false, err
	}

	var start time.Time
	switch {
	case !isZero(s.cursor.Object.ProgressTime):
		ts, err := parseBatchCursorTime(s.cursor.Object.ProgressTime)
		if err != nil {
			return objectBatchWindow{}, false, err
		}
		start = laterBatchStart(ts, s.cursor.Object.FirstEventTime, s.cursor.Object.LastEventTime)
	case !isZero(s.cursor.Object.FirstEventTime):
		// Legacy object collection resumed from first_event_time. Seed the first
		// batched window from that watermark so upgrades don't skip or replay data.
		ts, err := parseBatchCursorTime(s.cursor.Object.FirstEventTime)
		if err != nil {
			return objectBatchWindow{}, false, err
		}
		start = ts
	case !isZero(s.cursor.Object.LastEventTime):
		ts, err := parseBatchCursorTime(s.cursor.Object.LastEventTime)
		if err != nil {
			return objectBatchWindow{}, false, err
		}
		start = ts
	default:
		start = runEnd.Add(-batch.InitialInterval)
	}

	end := start.Add(batch.Window)
	if end.After(runEnd) {
		end = runEnd
	}
	if !end.After(start) {
		return objectBatchWindow{}, false, nil
	}

	return objectBatchWindow{
		Start: start,
		End:   end,
	}, true, nil
}

// laterBatchStart returns the latest of progressTS and any parseable legacy
// watermark in fallbacks. Unparseable fallbacks are ignored on purpose: the
// progress_time path is the authoritative cursor when set, so a corrupt
// legacy field shouldn't fail an otherwise valid resume — the symmetric
// projection in objectCursor (laterObjectResumeWatermark) makes the same
// trade-off for the unbatched direction.
func laterBatchStart(progressTS time.Time, fallbacks ...string) time.Time {
	start := progressTS
	for _, fallback := range fallbacks {
		if isZero(fallback) {
			continue
		}
		fallbackTS, err := parseBatchCursorTime(fallback)
		if err != nil {
			continue
		}
		if fallbackTS.After(start) {
			start = fallbackTS
		}
	}
	return start
}

// objectBatchConfig returns the Object method's batchConfig. It returns an
// error when either the event_monitoring_method.object block or its batch
// sub-block is nil, which should not happen once config validation has
// passed but is defended here because nextObjectBatchWindow and
// runObjectBatches both depend on a non-nil batchConfig.
func (s *salesforceInput) objectBatchConfig() (*batchConfig, error) {
	objectCfg, err := s.objectConfig()
	if err != nil {
		return nil, err
	}
	if objectCfg.Batch == nil {
		return nil, errors.New("internal error: object batch configuration is not set")
	}
	return objectCfg.Batch, nil
}
