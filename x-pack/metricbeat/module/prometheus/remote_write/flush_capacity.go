// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package remote_write

import (
	"sort"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// flushEventCapacity tracks histogram identities and bucket points represented by flush events
// awaiting (or retrying) publication. These counts participate in assembler admission limits.
type flushEventCapacity struct {
	histograms int
	buckets    int
}

func capacityFromFlushEvents(events map[string]mb.Event) flushEventCapacity {
	var total flushEventCapacity
	for _, ev := range events {
		total.addEvent(ev)
	}
	return total
}

func capacityFromFlushEvent(ev mb.Event) flushEventCapacity {
	var c flushEventCapacity
	c.addEvent(ev)
	return c
}

func (c *flushEventCapacity) addEvent(ev mb.Event) {
	if ev.ModuleFields == nil {
		return
	}
	for field, raw := range ev.ModuleFields {
		if field == "labels" {
			continue
		}
		metric, ok := raw.(mapstr.M)
		if !ok {
			continue
		}
		hist, ok := metric["histogram"].(mapstr.M)
		if !ok {
			continue
		}
		c.histograms++
		c.buckets += histogramBucketPoints(hist)
	}
}

func histogramBucketPoints(hist mapstr.M) int {
	switch v := hist["values"].(type) {
	case []float64:
		return len(v)
	case []any:
		return len(v)
	default:
		return 0
	}
}

func sortedEventKeys(events map[string]mb.Event) []string {
	keys := make([]string, 0, len(events))
	for k := range events {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// uint64FromCount converts a non-negative count (map length, capacity) to uint64.
// Negatives are clamped to zero; callers only pass lengths and capacity counters.
func uint64FromCount(n int) uint64 {
	if n <= 0 {
		return 0
	}
	return uint64(n) //nolint:gosec // G115: n is non-negative after the guard above
}
