// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tier_stats

import (
	"fmt"
	"slices"
	"sync"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// delta is the change between two scrapes.
type delta struct {
	count    int64
	duration float64
}

// state is what the previous scrape saw, which is needed because the histograms are cumulative.
type state struct {
	mu sync.Mutex
	// the previous scrape, which is replaced by every scrape, so anything that disappears is forgotten
	previous          snapshot
	previousTimestamp int64
}

var tierStatsState = &state{}

// compute converts the cumulative histograms into one document per tier and HTTP status code, which are the changes since the previous
// scrape that are added together across all of the tier's nodes and actions.
//
// A series is only counted when it has a previous value, so the first sample is ignored for a node (and for a status code that a node has
// not returned before), and a series whose count went backwards (the node restarted) is ignored. Every tier that has a node that was
// in the previous scrape gets a document for successful (2xx) requests even if there were none, so that quiet tiers report zeros
// rather than nothing, and the other status codes are only reported when there were requests.
func (s *state) compute(current snapshot, nowMillis int64) []mapstr.M {
	s.mu.Lock()
	defer s.mu.Unlock()

	previous := s.previous
	timestampDiff := nowMillis - s.previousTimestamp
	hasPrevious := s.previousTimestamp != 0 && previous.series != nil

	s.previous = current
	s.previousTimestamp = nowMillis

	if !hasPrevious || timestampDiff <= 0 {
		return nil
	}

	tiers := map[string]bool{}
	deltas := map[string]map[int]*delta{}

	for instance, tier := range current.instances {
		if _, known := previous.instances[instance]; known {
			tiers[tier] = true
		}
	}

	for key, value := range current.series {
		before, known := previous.series[key]

		// the first sample cannot be compared and a count that went backwards means that the node restarted
		if !known || value.count < before.count || value.count == before.count {
			continue
		}

		tier := current.instances[key.instance]
		statuses := deltas[tier]

		if statuses == nil {
			statuses = map[int]*delta{}
			deltas[tier] = statuses
		}

		change := statuses[key.status]

		if change == nil {
			change = &delta{}
			statuses[key.status] = change
		}

		change.count += value.count - before.count
		change.duration += max(value.sum-before.sum, 0)
	}

	return buildDocuments(tiers, deltas, timestampDiff)
}

// buildDocuments creates the documents in a stable order (by tier, then by status code).
func buildDocuments(tiers map[string]bool, deltas map[string]map[int]*delta, timestampDiff int64) []mapstr.M {
	sortedTiers := make([]string, 0, len(tiers))

	for tier := range tiers {
		sortedTiers = append(sortedTiers, tier)
	}

	slices.Sort(sortedTiers)

	docs := []mapstr.M{}

	for _, tier := range sortedTiers {
		statuses := make([]int, 0, len(deltas[tier])+1)
		hasSuccess := false

		for status := range deltas[tier] {
			statuses = append(statuses, status)
			hasSuccess = hasSuccess || isSuccess(status)
		}

		// a quiet tier still reports that it had no requests
		if !hasSuccess {
			statuses = append(statuses, 200)
		}

		slices.Sort(statuses)

		for _, status := range statuses {
			change := deltas[tier][status]

			if change == nil {
				change = &delta{}
			}

			docs = append(docs, document(tier, status, change, timestampDiff))
		}
	}

	return docs
}

func isSuccess(status int) bool {
	return status >= 200 && status <= 299
}

// document is the data for one tier and HTTP status code. Latency is only reported for successful requests because failing fast (or
// slowly) says nothing about how quickly the tier serves requests.
func document(tier string, status int, change *delta, timestampDiff int64) mapstr.M {
	doc := mapstr.M{
		"tier":                     tier,
		"status_code":              status,
		"status_group":             fmt.Sprintf("%dxx", status/100),
		"count_delta":              change.count,
		"duration_delta_in_millis": change.duration,
		"rate_per_second":          float64(change.count) / (float64(timestampDiff) / 1000),
		timestampDiffField:         timestampDiff,
	}

	if isSuccess(status) {
		latency := float64(0)

		if change.count > 0 {
			latency = change.duration / float64(change.count)
		}

		doc["latency_in_millis"] = latency
	}

	return doc
}
