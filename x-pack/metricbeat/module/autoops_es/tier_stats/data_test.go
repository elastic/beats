// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package tier_stats

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

func TestRequestBodyListsEveryMetricOnce(t *testing.T) {
	var body struct {
		Metrics []string `json:"metrics"`
	}

	require.NoError(t, json.Unmarshal(requestBody(), &body))

	// 10 REST actions and 7 async actions
	require.Len(t, body.Metrics, 17)
	require.Equal(t, requestedMetrics, body.Metrics)

	seen := map[string]bool{}

	for _, name := range body.Metrics {
		require.False(t, seen[name], "%s is requested twice", name)
		require.True(t, strings.HasPrefix(name, "es.rest.") || strings.HasPrefix(name, "es.async."), name)
		require.True(t, strings.HasSuffix(name, ".duration"), name)

		seen[name] = true
	}

	// the search tier
	for _, name := range []string{
		"es.rest.search.duration", "es.rest.msearch.duration", "es.rest.esql.duration", "es.rest.count.duration",
		"es.rest.search_scroll.duration", "es.rest.prometheus_query.duration",
		"es.async.esql.duration", "es.async.eql.duration", "es.async.sql.duration", "es.async.async_search.duration",
	} {
		require.True(t, seen[name], "%s is not requested", name)
	}

	// the indexing tier
	for _, name := range []string{
		"es.rest.index.duration", "es.rest.update.duration", "es.rest.bulk.duration", "es.rest.prometheus_write.duration",
		"es.async.reindex.duration", "es.async.update_by_query.duration", "es.async.delete_by_query.duration",
	} {
		require.True(t, seen[name], "%s is not requested", name)
	}
}

func TestResponseTolerance(t *testing.T) {
	// unknown fields, other metric types, and missing parts are all fine
	response, err := utils.DeserializeData[NodesMetricsResponse]([]byte(`{
		"_nodes": {"total": 3, "successful": 2, "failed": 1, "unknown": true},
		"cluster_name": "es",
		"resource_metrics": [
			{"resource_attributes": {"node.tier": "search", "service.instance.id": "a", "new.attribute": [1, 2]},
			 "metrics": {
				"es.rest.search.duration": {"type": "histogram", "unit": "ms", "measurements": [{"sum": 1.5, "count": 3, "attributes": {"http.response.status_code": 200, "other": "x"}}]},
				"es.some.gauge": {"type": "gauge", "measurements": [{"value": 1, "attributes": {}}]}
			 }},
			{"resource_attributes": {"node.tier": "index"}}
		],
		"missing_metrics": ["es.rest.count.duration"]
	}`))

	require.NoError(t, err)
	require.Len(t, response.ResourceMetrics, 2)

	measurement := response.ResourceMetrics[0].Metrics["es.rest.search.duration"].Measurements[0]

	require.InDelta(t, 1.5, measurement.Sum, 1e-9)
	require.EqualValues(t, 3, measurement.Count)

	// an unusable response is an error rather than a partial result
	_, err = utils.DeserializeData[NodesMetricsResponse]([]byte(`{"resource_metrics": "nope"}`))
	require.Error(t, err)
}

// seedPreviousScrape makes the next scrape be compared with a scrape that happened 10 seconds ago.
func seedPreviousScrape(t *testing.T, previous *NodesMetricsResponse) {
	t.Helper()

	tierStatsState.reset()

	require.Empty(t, tierStatsState.compute(readSnapshot(previous), time.Now().Add(-10*time.Second).UnixMilli()))
}

func TestEventsMappingReportsOneEventPerDocument(t *testing.T) {
	info := metricset.CreateClusterInfo("8.11.0")
	reporter := &mbtest.CapturingReporterV2{}

	seedPreviousScrape(t, response(
		node(searchNodeA, "search", hist(searchMetric, 200, 100, 1000), hist(searchMetric, 429, 1, 2)),
		node(indexNodeA, "index", hist(bulkMetric, 200, 10, 100)),
	))

	require.NoError(t, eventsMapping(reporter, &info, response(
		node(searchNodeA, "search", hist(searchMetric, 200, 200, 3000), hist(searchMetric, 429, 6, 12)),
		node(indexNodeA, "index", hist(bulkMetric, 200, 30, 500)),
	)))

	events := reporter.GetEvents()

	require.Empty(t, reporter.GetErrors())
	require.Len(t, events, 3)

	auto_ops_testing.CheckAllEventsUseSameTransactionId(t, events)

	for _, event := range events {
		auto_ops_testing.CheckEvent(t, event, info)
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.ModuleFields, "transaction_id"))
		// the milliseconds since the previous scrape is the root-level field that other documents have
		require.NotNil(t, auto_ops_testing.GetObjectValue(event.RootFields, "timestamp_diff"))
		require.NotContains(t, event.MetricSetFields, "timestamp_diff")
	}

	successful := 0

	for _, event := range events {
		// only what is read is reported, so the rate and the latency, which are calculated when reading, are not
		require.NotContains(t, event.MetricSetFields, "rate_per_second")
		require.NotContains(t, event.MetricSetFields, "latency_in_millis")

		if auto_ops_testing.GetObjectValue(event.MetricSetFields, "status_group") == "2xx" {
			require.EqualValues(t, 200, auto_ops_testing.GetObjectValue(event.MetricSetFields, "status_code"))
			successful++
		}
	}

	require.Equal(t, 2, successful)
}

func TestEventsMappingOnlyCountsTheNodesThatAnswered(t *testing.T) {
	info := metricset.CreateClusterInfo("8.11.0")
	reporter := &mbtest.CapturingReporterV2{}

	seedPreviousScrape(t, response(node(searchNodeA, "search", hist(searchMetric, 200, 100, 1000))))

	// the other search node did not answer, so it is not in the response
	partial := response(node(searchNodeA, "search", hist(searchMetric, 200, 150, 2000)))

	require.NoError(t, eventsMapping(reporter, &info, partial))

	// only the nodes that answered are counted, and what they served is still reported
	require.Len(t, reporter.GetEvents(), 1)
	require.EqualValues(t, 50, auto_ops_testing.GetObjectValue(reporter.GetEvents()[0].MetricSetFields, "count_delta"))
}

func TestEventsMappingReportsNothingForTheFirstScrape(t *testing.T) {
	info := metricset.CreateClusterInfo("8.11.0")
	reporter := &mbtest.CapturingReporterV2{}

	tierStatsState.reset()

	require.NoError(t, eventsMapping(reporter, &info, response(node(searchNodeA, "search", hist(searchMetric, 200, 100, 1000)))))
	require.Empty(t, reporter.GetEvents())
	require.Empty(t, reporter.GetErrors())
}

func TestStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		attributes map[string]any
		expected   int
		valid      bool
	}{
		{"number", map[string]any{statusCodeAttribute: float64(200)}, 200, true},
		{"numeric string", map[string]any{statusCodeAttribute: "429"}, 429, true},
		{"missing", map[string]any{}, 0, false},
		{"nil attributes", nil, 0, false},
		{"zero", map[string]any{statusCodeAttribute: float64(0)}, 0, false},
		{"negative", map[string]any{statusCodeAttribute: float64(-1)}, 0, false},
		{"text", map[string]any{statusCodeAttribute: "OK"}, 0, false},
		{"empty string", map[string]any{statusCodeAttribute: ""}, 0, false},
		{"boolean", map[string]any{statusCodeAttribute: true}, 0, false},
		{"null", map[string]any{statusCodeAttribute: nil}, 0, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, ok := statusCode(test.attributes)

			require.Equal(t, test.valid, ok)
			require.Equal(t, test.expected, code)
		})
	}
}

func TestStringAttribute(t *testing.T) {
	value, ok := stringAttribute(map[string]any{"a": "b"}, "a")
	require.True(t, ok)
	require.Equal(t, "b", value)

	for _, attributes := range []map[string]any{nil, {}, {"a": ""}, {"a": 1.0}, {"a": nil}} {
		_, ok = stringAttribute(attributes, "a")
		require.False(t, ok)
	}
}

func TestReadSnapshotReadsTheSeriesAndTheNodesWithTheirTiers(t *testing.T) {
	current := readSnapshot(response(
		node(searchNodeA, "search", hist(searchMetric, 200, 100, 1000), hist(searchMetric, 429, 5, 10), hist(msearchMetric, 200, 7, 70)),
		node(indexNodeA, "index", hist(bulkMetric, 201, 3, 30)),
	))

	require.Equal(t, map[string]string{searchNodeA: "search", indexNodeA: "index"}, current.instances)
	require.Equal(t, map[seriesKey]seriesValue{
		{searchNodeA, searchMetric, 200}:  {100, 1000},
		{searchNodeA, searchMetric, 429}:  {5, 10},
		{searchNodeA, msearchMetric, 200}: {7, 70},
		{indexNodeA, bulkMetric, 201}:     {3, 30},
	}, current.series)
}

func TestReadSnapshotAddsTheMeasurementsOfTheSameStatus(t *testing.T) {
	// the measurements only differ in attributes that are not used
	current := readSnapshot(response(node(searchNodeA, "search", hist(searchMetric, 200, 10, 100), hist(searchMetric, 200, 20, 400))))

	require.Equal(t, map[seriesKey]seriesValue{{searchNodeA, searchMetric, 200}: {30, 500}}, current.series)
}

func TestReadSnapshotIgnoresMeasurementsWithoutAValidStatusCode(t *testing.T) {
	current := readSnapshot(response(node(searchNodeA, "search",
		hist(searchMetric, nil, 1000, 1000),   // no status code at all
		hist(searchMetric, "abc", 1000, 1000), // not a number
		hist(searchMetric, 0, 1000, 1000),     // not a status code
		hist(searchMetric, -5, 1000, 1000),    // not a status code
		hist(searchMetric, true, 1000, 1000),  // not a number
		hist(searchMetric, []int{200}, 1000, 1000),
		hist(searchMetric, float64(200), 10, 100),
		hist(msearchMetric, "200", 5, 50),
	)))

	// numbers and numeric strings are both accepted; everything else is ignored
	require.Equal(t, map[seriesKey]seriesValue{
		{searchNodeA, searchMetric, 200}:  {10, 100},
		{searchNodeA, msearchMetric, 200}: {5, 50},
	}, current.series)
}

func TestReadSnapshotOnlyUsesTheRequestedHistograms(t *testing.T) {
	counter := hist(searchMetric, 200, 100, 100)
	counter.kind = "counter"

	current := readSnapshot(response(node(searchNodeA, "search", hist("es.rest.unrequested.duration", 200, 10, 10), counter)))

	// an unrequested metric and a metric that is not a histogram are not used, but the node is still known
	require.Empty(t, current.series)
	require.Equal(t, map[string]string{searchNodeA: "search"}, current.instances)
}

func TestReadSnapshotIgnoresNodesWithoutATierOrAnInstance(t *testing.T) {
	withoutTier := node(searchNodeB, "search", hist(searchMetric, 200, 100, 100))
	delete(withoutTier.ResourceAttributes, nodeTierAttribute)

	withoutInstance := node("", "search", hist(searchMetric, 200, 100, 100))
	delete(withoutInstance.ResourceAttributes, serviceInstanceIDAttribute)

	emptyTier := node(indexNodeA, "", hist(bulkMetric, 200, 100, 100))

	current := readSnapshot(response(node(searchNodeA, "search", hist(searchMetric, 200, 5, 5)), withoutTier, withoutInstance, emptyTier))

	require.Equal(t, map[string]string{searchNodeA: "search"}, current.instances)
	require.Equal(t, map[seriesKey]seriesValue{{searchNodeA, searchMetric, 200}: {5, 5}}, current.series)
}

func TestReadSnapshotKnowsTheNodesThatHaveNoMetrics(t *testing.T) {
	// a project that has never received a request returns its nodes without metrics
	current := readSnapshot(response(node(searchNodeA, "search"), node(indexNodeA, "index")))

	require.Empty(t, current.series)
	require.Equal(t, map[string]string{searchNodeA: "search", indexNodeA: "index"}, current.instances)
}

func TestReadSnapshotOfAnEmptyResponseIsEmptyButNotNil(t *testing.T) {
	current := readSnapshot(response())

	// it is something that was seen (nothing), which is not the same as never having seen anything
	require.NotNil(t, current.series)
	require.NotNil(t, current.instances)
	require.Empty(t, current.series)
	require.Empty(t, current.instances)
}
