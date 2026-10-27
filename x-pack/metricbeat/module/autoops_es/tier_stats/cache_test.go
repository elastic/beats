// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package tier_stats

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	searchMetric  = "es.rest.search.duration"
	msearchMetric = "es.rest.msearch.duration"
	asyncEsql     = "es.async.esql.duration"
	bulkMetric    = "es.rest.bulk.duration"

	searchNodeA = "es-es-search-a"
	searchNodeB = "es-es-search-b"
	indexNodeA  = "es-es-index-a"

	start = int64(1_000_000)
	tick  = int64(10_000)
)

// reset forgets everything that was seen, so that the next scrape is treated as the first one.
func (s *state) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.previous = snapshot{}
	s.previousTimestamp = 0
}

// sample is one measurement of a node's histogram, which is cumulative.
type sample struct {
	metric string
	// the HTTP status code, which is left out when nil
	status any
	count  int64
	sum    float64
	// the type of the metric (a histogram unless set)
	kind string
}

func hist(metric string, status any, count int64, sum float64) sample {
	return sample{metric: metric, status: status, count: count, sum: sum}
}

func node(instance string, tier string, samples ...sample) ResourceMetrics {
	metrics := map[string]Metric{}

	for _, s := range samples {
		metric := metrics[s.metric]
		metric.Type = histogramType

		if s.kind != "" {
			metric.Type = s.kind
		}

		attributes := map[string]any{}

		if s.status != nil {
			status := s.status

			// JSON numbers are decoded as float64
			if code, isInt := status.(int); isInt {
				status = float64(code)
			}

			attributes[statusCodeAttribute] = status
		}

		metric.Measurements = append(metric.Measurements, Measurement{Count: s.count, Sum: s.sum, Attributes: attributes})
		metrics[s.metric] = metric
	}

	return ResourceMetrics{
		ResourceAttributes: map[string]any{serviceInstanceIDAttribute: instance, nodeTierAttribute: tier},
		Metrics:            metrics,
	}
}

func response(nodes ...ResourceMetrics) *NodesMetricsResponse {
	return &NodesMetricsResponse{ResourceMetrics: nodes}
}

// scrape computes the documents for a response, as if it was received at the given time.
func scrape(s *state, at int64, nodes ...ResourceMetrics) []mapstr.M {
	return s.compute(readSnapshot(response(nodes...)), at)
}

// find gets the only document for the tier and status code, or fails.
func find(t *testing.T, docs []mapstr.M, tier string, status int) mapstr.M {
	t.Helper()

	var found []mapstr.M

	for _, doc := range docs {
		if doc["tier"] == tier && doc["status_code"] == status {
			found = append(found, doc)
		}
	}

	require.Len(t, found, 1, "expected exactly one document for %s %d in %v", tier, status, docs)

	return found[0]
}

func requireDocument(t *testing.T, doc mapstr.M, count int64, duration float64) {
	t.Helper()

	require.EqualValues(t, count, doc["count_delta"])
	require.InDelta(t, duration, doc["duration_delta_in_millis"], 1e-9)
}

func TestFirstScrapeProducesNothing(t *testing.T) {
	s := &state{}

	require.Empty(t, scrape(s, start, node(searchNodeA, "search", hist(searchMetric, 200, 100, 5000))))
}

func TestDeltaRateAndLatency(t *testing.T) {
	s := &state{}

	scrape(s, start, node(searchNodeA, "search", hist(searchMetric, 200, 100, 5000)))
	docs := scrape(s, start+tick, node(searchNodeA, "search", hist(searchMetric, 200, 250, 12500)))

	require.Len(t, docs, 1)

	doc := docs[0]

	require.Equal(t, "search", doc["tier"])
	require.Equal(t, 200, doc["status_code"])
	require.Equal(t, "2xx", doc["status_group"])
	requireDocument(t, doc, 150, 7500)
	require.InDelta(t, 15.0, doc["rate_per_second"], 1e-9)
	require.InDelta(t, 50.0, doc["latency_in_millis"], 1e-9)
	require.EqualValues(t, tick, doc["timestamp_diff"])
}

func TestRateUsesTheRealInterval(t *testing.T) {
	s := &state{}

	scrape(s, start, node(searchNodeA, "search", hist(searchMetric, 200, 0, 0)))
	docs := scrape(s, start+30_001, node(searchNodeA, "search", hist(searchMetric, 200, 3000, 6000)))

	doc := find(t, docs, "search", 200)

	require.EqualValues(t, 30_001, doc["timestamp_diff"])
	require.InDelta(t, 3000/30.001, doc["rate_per_second"], 1e-9)
}

func TestNodesAndActionsAreAddedTogether(t *testing.T) {
	s := &state{}

	scrape(s, start,
		node(searchNodeA, "search", hist(searchMetric, 200, 1000, 10_000), hist(msearchMetric, 200, 500, 5000)),
		node(searchNodeB, "search", hist(searchMetric, 200, 2000, 8000), hist(asyncEsql, 200, 10, 20_000)),
	)
	docs := scrape(s, start+tick,
		node(searchNodeA, "search", hist(searchMetric, 200, 1100, 10_500), hist(msearchMetric, 200, 550, 5250)),
		node(searchNodeB, "search", hist(searchMetric, 200, 2100, 9000), hist(asyncEsql, 200, 11, 22_000)),
	)

	// one document per tier and status, not one per node or action
	require.Len(t, docs, 1)
	// 100 + 50 + 100 + 1 requests taking 500 + 250 + 1000 + 2000 ms
	requireDocument(t, docs[0], 251, 3750)
	require.InDelta(t, 3750.0/251, docs[0]["latency_in_millis"], 1e-9)
}

func TestTiersAreSeparate(t *testing.T) {
	s := &state{}

	scrape(s, start, node(searchNodeA, "search", hist(searchMetric, 200, 10, 100)), node(indexNodeA, "index", hist(bulkMetric, 200, 20, 400)))
	docs := scrape(s, start+tick, node(searchNodeA, "search", hist(searchMetric, 200, 20, 300)), node(indexNodeA, "index", hist(bulkMetric, 200, 25, 700)))

	require.Len(t, docs, 2)
	requireDocument(t, find(t, docs, "search", 200), 10, 200)
	requireDocument(t, find(t, docs, "index", 200), 5, 300)
}

func TestSuccessStatusCodesAreSeparateDocumentsWithLatency(t *testing.T) {
	s := &state{}

	scrape(s, start, node(indexNodeA, "index", hist(bulkMetric, 200, 0, 0), hist(bulkMetric, 201, 0, 0)))
	docs := scrape(s, start+tick, node(indexNodeA, "index", hist(bulkMetric, 200, 10, 100), hist(bulkMetric, 201, 4, 80)))

	require.Len(t, docs, 2)

	for status, latency := range map[int]float64{200: 10, 201: 20} {
		doc := find(t, docs, "index", status)

		require.Equal(t, "2xx", doc["status_group"])
		require.InDelta(t, latency, doc["latency_in_millis"], 1e-9)
	}
}

func TestErrorStatusCodesHaveNoLatency(t *testing.T) {
	s := &state{}

	scrape(s, start, node(searchNodeA, "search",
		hist(searchMetric, 200, 0, 0), hist(searchMetric, 404, 0, 0), hist(searchMetric, 429, 0, 0), hist(searchMetric, 500, 0, 0)))
	docs := scrape(s, start+tick, node(searchNodeA, "search",
		hist(searchMetric, 200, 10, 100), hist(searchMetric, 404, 1, 1), hist(searchMetric, 429, 7, 14), hist(searchMetric, 500, 2, 6000)))

	require.Len(t, docs, 4)

	for status, group := range map[int]string{404: "4xx", 429: "4xx", 500: "5xx"} {
		doc := find(t, docs, "search", status)

		require.Equal(t, group, doc["status_group"])
		require.NotContains(t, doc, "latency_in_millis")
		require.Contains(t, doc, "rate_per_second")
	}

	// the slow errors do not affect the latency of the successful requests
	require.InDelta(t, 10.0, find(t, docs, "search", 200)["latency_in_millis"], 1e-9)
}

func TestQuietTierReportsZeros(t *testing.T) {
	s := &state{}

	// no change at all, including for the error series
	scrape(s, start, node(searchNodeA, "search", hist(searchMetric, 200, 100, 5000), hist(searchMetric, 500, 2, 40)))
	docs := scrape(s, start+tick, node(searchNodeA, "search", hist(searchMetric, 200, 100, 5000), hist(searchMetric, 500, 2, 40)))

	require.Len(t, docs, 1)

	doc := find(t, docs, "search", 200)

	requireDocument(t, doc, 0, 0)
	require.InDelta(t, 0.0, doc["rate_per_second"], 1e-9)
	require.InDelta(t, 0.0, doc["latency_in_millis"], 1e-9)
}

func TestTierWithoutAnyMetricsReportsZeros(t *testing.T) {
	s := &state{}

	// a project that has never received a request returns nodes without metrics
	scrape(s, start, node(searchNodeA, "search"), node(indexNodeA, "index"))
	docs := scrape(s, start+tick, node(searchNodeA, "search"), node(indexNodeA, "index"))

	require.Len(t, docs, 2)

	for _, tier := range []string{"index", "search"} {
		doc := find(t, docs, tier, 200)

		requireDocument(t, doc, 0, 0)
		require.InDelta(t, 0.0, doc["latency_in_millis"], 1e-9)
	}
}

func TestOnlyFailuresStillReportSuccessZeros(t *testing.T) {
	s := &state{}

	scrape(s, start, node(searchNodeA, "search", hist(searchMetric, 200, 100, 5000), hist(searchMetric, 500, 0, 0)))
	docs := scrape(s, start+tick, node(searchNodeA, "search", hist(searchMetric, 200, 100, 5000), hist(searchMetric, 500, 5, 50)))

	require.Len(t, docs, 2)
	requireDocument(t, find(t, docs, "search", 200), 0, 0)
	requireDocument(t, find(t, docs, "search", 500), 5, 50)
}

func TestSuccessfulRequestsAreNotReportedAsQuietWhenOnlyNon200Succeeds(t *testing.T) {
	s := &state{}

	scrape(s, start, node(indexNodeA, "index", hist(bulkMetric, 201, 0, 0)))
	docs := scrape(s, start+tick, node(indexNodeA, "index", hist(bulkMetric, 201, 3, 30)))

	// the created documents are the successful requests, so there is no zero 200 document next to them
	require.Len(t, docs, 1)
	requireDocument(t, find(t, docs, "index", 201), 3, 30)
}

func TestCounterResetIsIgnored(t *testing.T) {
	s := &state{}

	scrape(s, start, node(searchNodeA, "search", hist(searchMetric, 200, 1000, 50_000)), node(searchNodeB, "search", hist(searchMetric, 200, 500, 5000)))
	// node A restarted, so its counters went back to a small number
	docs := scrape(s, start+tick, node(searchNodeA, "search", hist(searchMetric, 200, 12, 600)), node(searchNodeB, "search", hist(searchMetric, 200, 540, 5400)))

	require.Len(t, docs, 1)
	// only node B counts, and nothing is negative
	requireDocument(t, docs[0], 40, 400)

	// the restarted node counts again from its new baseline
	docs = scrape(s, start+2*tick, node(searchNodeA, "search", hist(searchMetric, 200, 22, 1100)), node(searchNodeB, "search", hist(searchMetric, 200, 540, 5400)))

	requireDocument(t, find(t, docs, "search", 200), 10, 500)
}

func TestResetOfTheOnlyNodeReportsZeros(t *testing.T) {
	s := &state{}

	scrape(s, start, node(searchNodeA, "search", hist(searchMetric, 200, 1000, 50_000)))
	docs := scrape(s, start+tick, node(searchNodeA, "search", hist(searchMetric, 200, 3, 30)))

	require.Len(t, docs, 1)
	requireDocument(t, docs[0], 0, 0)
	require.GreaterOrEqual(t, docs[0]["rate_per_second"], 0.0)
}

func TestSumGoingDownNeverMakesTheDurationNegative(t *testing.T) {
	s := &state{}

	scrape(s, start, node(searchNodeA, "search", hist(searchMetric, 200, 10, 1000)))
	docs := scrape(s, start+tick, node(searchNodeA, "search", hist(searchMetric, 200, 20, 900)))

	requireDocument(t, docs[0], 10, 0)
	require.InDelta(t, 0.0, docs[0]["latency_in_millis"], 1e-9)
}

func TestNewNodeIsIgnoredUntilItHasABaseline(t *testing.T) {
	s := &state{}

	scrape(s, start, node(searchNodeA, "search", hist(searchMetric, 200, 100, 1000)))
	// node B joins with a large cumulative count that must not be counted as new requests
	docs := scrape(s, start+tick, node(searchNodeA, "search", hist(searchMetric, 200, 110, 1100)), node(searchNodeB, "search", hist(searchMetric, 200, 90_000, 900_000)))

	requireDocument(t, find(t, docs, "search", 200), 10, 100)

	docs = scrape(s, start+2*tick, node(searchNodeA, "search", hist(searchMetric, 200, 120, 1200)), node(searchNodeB, "search", hist(searchMetric, 200, 90_020, 900_400)))

	requireDocument(t, find(t, docs, "search", 200), 30, 500)
}

func TestNewStatusCodeOnKnownNodeIsIgnoredOnce(t *testing.T) {
	s := &state{}

	scrape(s, start, node(searchNodeA, "search", hist(searchMetric, 200, 100, 1000)))
	// the first 429 ever returned by the node: there is no previous value to compare it with
	docs := scrape(s, start+tick, node(searchNodeA, "search", hist(searchMetric, 200, 110, 1100), hist(searchMetric, 429, 5, 10)))

	require.Len(t, docs, 1)
	requireDocument(t, find(t, docs, "search", 200), 10, 100)

	docs = scrape(s, start+2*tick, node(searchNodeA, "search", hist(searchMetric, 200, 120, 1200), hist(searchMetric, 429, 9, 18)))

	require.Len(t, docs, 2)
	requireDocument(t, find(t, docs, "search", 429), 4, 8)
}

func TestNodeThatDisappearsIsForgotten(t *testing.T) {
	s := &state{}

	scrape(s, start, node(searchNodeA, "search", hist(searchMetric, 200, 100, 1000)), node(searchNodeB, "search", hist(searchMetric, 200, 100, 1000)))
	// node B does not answer
	docs := scrape(s, start+tick, node(searchNodeA, "search", hist(searchMetric, 200, 110, 1100)))

	requireDocument(t, find(t, docs, "search", 200), 10, 100)

	// node B is back with everything it served in the meantime, which cannot be attributed to this scrape
	docs = scrape(s, start+2*tick, node(searchNodeA, "search", hist(searchMetric, 200, 120, 1200)), node(searchNodeB, "search", hist(searchMetric, 200, 5000, 50_000)))

	requireDocument(t, find(t, docs, "search", 200), 10, 100)
}

func TestNewTierIsNotReportedUntilItHasABaseline(t *testing.T) {
	s := &state{}

	scrape(s, start, node(searchNodeA, "search", hist(searchMetric, 200, 100, 1000)))
	// the indexing tier shows up later, and must not report an (incorrect) zero for a scrape it was not compared in
	docs := scrape(s, start+tick, node(searchNodeA, "search", hist(searchMetric, 200, 110, 1100)), node(indexNodeA, "index", hist(bulkMetric, 200, 50, 500)))

	require.Len(t, docs, 1)
	require.Equal(t, "search", docs[0]["tier"])

	docs = scrape(s, start+2*tick, node(searchNodeA, "search", hist(searchMetric, 200, 120, 1200)), node(indexNodeA, "index", hist(bulkMetric, 200, 60, 700)))

	require.Len(t, docs, 2)
	requireDocument(t, find(t, docs, "index", 200), 10, 200)
}

func TestClockThatDoesNotAdvanceProducesNothingAndRecovers(t *testing.T) {
	s := &state{}

	scrape(s, start, node(searchNodeA, "search", hist(searchMetric, 200, 100, 1000)))
	// same (or an earlier) time cannot produce a rate
	require.Empty(t, scrape(s, start, node(searchNodeA, "search", hist(searchMetric, 200, 150, 1500))))
	require.Empty(t, scrape(s, start-5, node(searchNodeA, "search", hist(searchMetric, 200, 160, 1600))))

	// the next scrape compares with the last one that was seen
	docs := scrape(s, start+tick, node(searchNodeA, "search", hist(searchMetric, 200, 170, 1700)))

	requireDocument(t, docs[0], 10, 100)
}

func TestEmptyResponseStartsOver(t *testing.T) {
	s := &state{}

	scrape(s, start, node(searchNodeA, "search", hist(searchMetric, 200, 100, 1000)))
	require.Empty(t, scrape(s, start+tick))
	// the nodes are back, but they were not compared with anything
	require.Empty(t, scrape(s, start+2*tick, node(searchNodeA, "search", hist(searchMetric, 200, 200, 2000))))

	docs := scrape(s, start+3*tick, node(searchNodeA, "search", hist(searchMetric, 200, 210, 2100)))

	requireDocument(t, docs[0], 10, 100)
}

func TestResetForgetsEverything(t *testing.T) {
	s := &state{}

	scrape(s, start, node(searchNodeA, "search", hist(searchMetric, 200, 100, 1000)))
	s.reset()

	require.Empty(t, scrape(s, start+tick, node(searchNodeA, "search", hist(searchMetric, 200, 200, 2000))))
}

func TestDocumentsAreInAStableOrder(t *testing.T) {
	s := &state{}

	nodes := func(base int64) []ResourceMetrics {
		return []ResourceMetrics{
			node(searchNodeA, "search", hist(searchMetric, 500, base, 0), hist(searchMetric, 200, base, 0), hist(searchMetric, 404, base, 0)),
			node(indexNodeA, "index", hist(bulkMetric, 429, base, 0), hist(bulkMetric, 201, base, 0)),
		}
	}

	scrape(s, start, nodes(0)...)
	docs := scrape(s, start+tick, nodes(5)...)

	var order [][2]any

	for _, doc := range docs {
		order = append(order, [2]any{doc["tier"], doc["status_code"]})
	}

	// tiers alphabetically, then status codes ascending
	require.Equal(t, [][2]any{{"index", 201}, {"index", 429}, {"search", 200}, {"search", 404}, {"search", 500}}, order)
}
