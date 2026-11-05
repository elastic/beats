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
// Every status code that a tier returns gets a document in every scrape. The first one for a status code (the first scrape, or the first
// time that the tier returns the status code) has no count_delta or duration_delta, because there is no previous scrape to subtract
// them from, but it still reports that the status code was returned.
//
// Within a status code that was already returned, a node's series is only counted when it has a previous value, so a node that was not in
// the previous scrape is ignored once, and a series whose count went backwards (the node restarted) is ignored. Every tier that has a
// node that was in the previous scrape gets a document for successful (2xx) requests even if there were none, so that quiet tiers
// report zeros rather than nothing, and the other status codes are only reported when there were requests.
func (s *state) compute(current snapshot, nowMillis int64) []mapstr.M {
	s.mu.Lock()
	defer s.mu.Unlock()

	previous := s.previous
	hasPreviousScrape := s.previousTimestamp != 0
	timestampDiff := nowMillis - s.previousTimestamp

	s.previous = current
	s.previousTimestamp = nowMillis

	if hasPreviousScrape && timestampDiff <= 0 {
		return nil
	}

	// what each tier returned before, to know which status codes are new
	returnedBefore := map[string]map[int]bool{}

	for key := range previous.series {
		tier := previous.instances[key.instance]

		if returnedBefore[tier] == nil {
			returnedBefore[tier] = map[int]bool{}
		}

		returnedBefore[tier][key.status] = true
	}

	tiers := map[string]bool{}
	// a nil delta is a status code that is new, which has nothing to subtract from
	deltas := map[string]map[int]*delta{}

	for instance, tier := range current.instances {
		if _, known := previous.instances[instance]; known {
			tiers[tier] = true
		}
	}

	for key, value := range current.series {
		tier := current.instances[key.instance]
		statuses := deltas[tier]

		if statuses == nil {
			statuses = map[int]*delta{}
			deltas[tier] = statuses
		}

		if !returnedBefore[tier][key.status] {
			tiers[tier] = true

			if _, seen := statuses[key.status]; !seen {
				statuses[key.status] = nil
			}

			continue
		}

		before, known := previous.series[key]

		// a node that is new for a status code that was already returned cannot be compared, a count that went backwards means that the
		// node restarted, and the same count means that there were no requests
		if !known || value.count <= before.count {
			continue
		}

		change := statuses[key.status]

		if change == nil {
			change = &delta{}
			statuses[key.status] = change
		}

		change.count += value.count - before.count
		change.duration += max(value.sum-before.sum, 0)
	}

	return buildDocuments(tiers, deltas, timestampDiff, hasPreviousScrape)
}

// buildDocuments creates the documents in a stable order (by tier, then by status code).
func buildDocuments(tiers map[string]bool, deltas map[string]map[int]*delta, timestampDiff int64, hasPreviousScrape bool) []mapstr.M {
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

		// a quiet tier still reports that it had no requests, but only when there is a previous scrape to compare it with
		if !hasSuccess && hasPreviousScrape {
			statuses = append(statuses, 200)
		}

		slices.Sort(statuses)

		for _, status := range statuses {
			change, hasChange := deltas[tier][status]

			if !hasChange {
				change = &delta{}
			}

			docs = append(docs, document(tier, status, change, timestampDiff, hasPreviousScrape))
		}
	}

	return docs
}

func isSuccess(status int) bool {
	return status >= 200 && status <= 299
}

// document is the data for one tier and HTTP status code. The rate and the latency are not reported: they are calculated from the
// count and the duration of all the scrapes in the period that is being read, which cannot be done from per scrape values. A nil change
// is a status code that was not returned before, which has no count or duration.
func document(tier string, status int, change *delta, timestampDiff int64, hasPreviousScrape bool) mapstr.M {
	doc := mapstr.M{
		"tier":         tier,
		"status_code":  status,
		"status_group": fmt.Sprintf("%dxx", status/100),
	}

	if change != nil {
		doc["count_delta"] = change.count
		doc["duration_delta_in_millis"] = change.duration
	}

	if hasPreviousScrape {
		doc[timestampDiffField] = timestampDiff
	}

	return doc
}
