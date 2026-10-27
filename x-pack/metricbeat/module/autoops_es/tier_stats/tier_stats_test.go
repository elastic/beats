// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package tier_stats

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var useNamedMetricSet = auto_ops_testing.UseNamedMetricSet(TierStatsMetricSet)

const (
	// a serverless cluster
	clusterInfoFile = "../_meta/test/cluster_info.8.11.0.json"
	// a cluster that is not serverless
	otherClusterInfoFile = "../_meta/test/cluster_info.8.15.3.json"
)

// request is what the server received.
type request struct {
	method      string
	uri         string
	contentType string
	body        string
}

// testServer serves the cluster info and `_nodes/metrics`, and records what it was asked for.
type testServer struct {
	*httptest.Server

	mu       sync.Mutex
	requests []request
	// serves the _nodes/metrics route
	metrics http.HandlerFunc
}

func newTestServer(t *testing.T, clusterInfo []byte, metrics http.HandlerFunc) *testServer {
	t.Helper()

	s := &testServer{metrics: metrics}

	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		// the route's handler reads the body as well
		r.Body = io.NopCloser(bytes.NewReader(body))

		s.mu.Lock()
		s.requests = append(s.requests, request{method: r.Method, uri: r.RequestURI, contentType: r.Header.Get("Content-Type"), body: string(body)})
		s.mu.Unlock()

		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(clusterInfo)
		case "/_nodes/metrics":
			s.metrics(w, r)
		default:
			t.Errorf("unrecognized request %v %v", r.Method, r.RequestURI)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))

	t.Cleanup(s.Close)

	return s
}

func (s *testServer) recorded() []request {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]request(nil), s.requests...)
}

func serve(data []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	return data
}

func fetchOnce(t *testing.T, server *testServer) (*mbtest.CapturingReporterV2, error) {
	t.Helper()

	ms := mbtest.NewReportingMetricSetV2Error(t, useNamedMetricSet(server.Server))
	reporter := &mbtest.CapturingReporterV2{}

	return reporter, ms.Fetch(reporter)
}

// errorEvents gets the events that report an error rather than data.
func errorEvents(events []mb.Event) []mb.Event {
	var errors []mb.Event

	for _, event := range events {
		if _, isError := event.RootFields["error"]; isError {
			errors = append(errors, event)
		}
	}

	return errors
}

// The request is a POST of the metrics to the route, which is checked for every request that the fixture tests make.
func serveFixture(t *testing.T) func(t *testing.T, clusterInfo []byte, data []byte, version string) *httptest.Server {
	return func(t *testing.T, clusterInfo []byte, data []byte, _ string) *httptest.Server {
		return newTestServer(t, clusterInfo, func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)

			// assert (rather than require) because this runs outside of the test's goroutine
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "timeout=5s", r.URL.RawQuery)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var requested struct {
				Metrics []string `json:"metrics"`
			}

			assert.NoError(t, json.Unmarshal(body, &requested))
			assert.Equal(t, requestedMetrics, requested.Metrics)

			serve(data)(w, r)
		}).Server
	}
}

// Everything that changed between the baseline and the fixtures.
type change struct {
	count int64
	sum   float64
}

type seriesID struct {
	instance string
	metric   string
	status   int
}

const (
	fixtureIndexNode   = "es-es-index-6865646ffb-br5jn"
	fixtureSearchNodeA = "es-es-search-5d4d754484-d6hcl"
	fixtureSearchNodeB = "es-es-search-5d4d754484-zq8mw"
)

// What each node served since the previous scrape of the main fixture.
var fixtureChanges = map[seriesID]change{
	{fixtureIndexNode, "es.rest.bulk.duration", 200}:      {200, 30_000},
	{fixtureIndexNode, "es.rest.index.duration", 200}:     {20, 2000},
	{fixtureIndexNode, "es.rest.update.duration", 201}:    {2, 233.379006},
	{fixtureIndexNode, "es.rest.update.duration", 200}:    {10, 1000},
	{fixtureIndexNode, "es.async.reindex.duration", 200}:  {1, 3000},
	{fixtureSearchNodeA, "es.rest.esql.duration", 200}:    {50, 500},
	{fixtureSearchNodeA, "es.rest.search.duration", 200}:  {100, 400},
	{fixtureSearchNodeA, "es.rest.msearch.duration", 200}: {30, 90},
	{fixtureSearchNodeA, "es.rest.search.duration", 404}:  {5, 5},
	{fixtureSearchNodeA, "es.rest.search.duration", 429}:  {10, 20},
	{fixtureSearchNodeA, "es.rest.search.duration", 500}:  {3, 3000},
	{fixtureSearchNodeB, "es.rest.search.duration", 200}:  {100, 300},
	{fixtureSearchNodeB, "es.async.esql.duration", 200}:   {1, 2000},
}

// baselineFor seeds the state as if the fixture's nodes were scraped 10 seconds ago, with `changes` fewer requests than the fixture.
func baselineFor(t *testing.T, fixture NodesMetricsResponse, changes map[seriesID]change) {
	t.Helper()

	// work on a copy so that the fixture's data is not changed
	var baseline NodesMetricsResponse

	encoded, err := json.Marshal(fixture)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(encoded, &baseline))

	for _, node := range baseline.ResourceMetrics {
		instance, _ := stringAttribute(node.ResourceAttributes, serviceInstanceIDAttribute)

		for name, metric := range node.Metrics {
			for i, measurement := range metric.Measurements {
				status, ok := statusCode(measurement.Attributes)

				if !ok {
					continue
				}

				if c, found := changes[seriesID{instance, name, status}]; found {
					metric.Measurements[i].Count -= c.count
					metric.Measurements[i].Sum -= c.sum
				}
			}
		}
	}

	seedPreviousScrape(t, &baseline)
}

func documents(events []mb.Event) []mapstr.M {
	docs := make([]mapstr.M, 0, len(events))

	for _, event := range events {
		docs = append(docs, event.MetricSetFields)
	}

	return docs
}

func requireTimestampDiff(t *testing.T, docs []mapstr.M) {
	t.Helper()

	for _, doc := range docs {
		// the baseline was taken 10 seconds ago
		require.Greater(t, doc["timestamp_diff"], int64(9_999))
		require.Less(t, doc["timestamp_diff"], int64(20_000))
	}
}

func TestSuccessfulFetchWithFixtures(t *testing.T) {
	metricset.RunTestsForMetricSetWithGlobFiles(t, "./_meta/test/*tier_stats.*.json", serveFixture(t), useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[NodesMetricsResponse]) {
		baselineFor(t, data.Data, fixtureChanges)

		require.NoError(t, data.MetricSet.Fetch(data.Reporter))

		events := data.Reporter.GetEvents()
		docs := documents(events)

		require.Empty(t, data.Reporter.GetErrors())
		require.Empty(t, errorEvents(events))
		auto_ops_testing.CheckAllEventsUseSameTransactionId(t, events)

		for _, event := range events {
			auto_ops_testing.CheckEvent(t, event, data.ClusterInfo)
		}

		requireTimestampDiff(t, docs)

		switch filepath.Base(data.File) {
		case "tier_stats.8.11.0.json":
			// the index tier: the bulk, index, update (200) and reindex (async) requests; and the created documents
			require.Len(t, docs, 6)
			requireDocument(t, find(t, docs, "index", 200), 231, 36_000)
			require.InDelta(t, 36_000.0/231, find(t, docs, "index", 200)["latency_in_millis"], 1e-9)
			requireDocument(t, find(t, docs, "index", 201), 2, 233.379006)

			// the search tier: both nodes, and the REST and async requests; the unrequested histogram, the counter, and the measurement
			// without a status code are not used
			requireDocument(t, find(t, docs, "search", 200), 281, 3290)
			require.InDelta(t, 3290.0/281, find(t, docs, "search", 200)["latency_in_millis"], 1e-9)
			requireDocument(t, find(t, docs, "search", 404), 5, 5)
			requireDocument(t, find(t, docs, "search", 429), 10, 20)
			requireDocument(t, find(t, docs, "search", 500), 3, 3000)

			for _, status := range []int{404, 429, 500} {
				require.NotContains(t, find(t, docs, "search", status), "latency_in_millis")
			}

			for _, doc := range docs {
				require.InDelta(t, float64(doc["count_delta"].(int64))/(float64(doc["timestamp_diff"].(int64))/1000), doc["rate_per_second"], 1e-9)
			}
		case "no_traffic.tier_stats.8.11.0.json":
			// the nodes have never served a request, which is a rate and a latency of 0 rather than no data
			require.Len(t, docs, 2)

			for _, tier := range []string{"index", "search"} {
				doc := find(t, docs, tier, 200)

				requireDocument(t, doc, 0, 0)
				require.InDelta(t, 0.0, doc["rate_per_second"], 1e-9)
				require.InDelta(t, 0.0, doc["latency_in_millis"], 1e-9)
			}
		case "partial.tier_stats.8.11.0.json":
			// the second search node did not answer, so only the nodes that did are counted
			require.Len(t, docs, 6)
			requireDocument(t, find(t, docs, "search", 200), 180, 990)
			requireDocument(t, find(t, docs, "index", 200), 231, 36_000)
		default:
			t.Fatalf("no expectations for %v", data.File)
		}
	})
}

func TestFetchSendsTheRequestAndOnlyTheClusterInfoRequestIsAGet(t *testing.T) {
	fixture := readFile(t, "./_meta/test/no_traffic.tier_stats.8.11.0.json")
	server := newTestServer(t, readFile(t, clusterInfoFile), serve(fixture))

	ms := mbtest.NewReportingMetricSetV2Error(t, useNamedMetricSet(server.Server))

	tierStatsState.reset()

	first := &mbtest.CapturingReporterV2{}
	require.NoError(t, ms.Fetch(first))
	// the first scrape has nothing to compare with
	require.Empty(t, first.GetEvents())

	// make the next scrape look like it happened 10 seconds later
	tierStatsState.mu.Lock()
	tierStatsState.previousTimestamp -= 10_000
	tierStatsState.mu.Unlock()

	second := &mbtest.CapturingReporterV2{}
	require.NoError(t, ms.Fetch(second))
	require.Len(t, second.GetEvents(), 2)

	requests := server.recorded()

	require.Len(t, requests, 4)

	// each scrape asks for the cluster info and then the metrics, and the cluster info request is not affected by the previous request
	for i := 0; i < len(requests); i += 2 {
		require.Equal(t, http.MethodGet, requests[i].method)
		require.Equal(t, "/", requests[i].uri)
		require.Empty(t, requests[i].body)

		require.Equal(t, http.MethodPost, requests[i+1].method)
		require.Equal(t, TierStatsPath, requests[i+1].uri)
		require.Equal(t, "application/json", requests[i+1].contentType)
		require.JSONEq(t, string(requestBody()), requests[i+1].body)
	}
}

func TestFetchOtherFailuresAreReported(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		code    string
	}{
		{"server error", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"type":"exception","reason":"boom"},"status":500}`))
		}, "HTTP_500"},
		{"too many requests", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}, "HTTP_429"},
		// the API not being there (yet) is reported like any other failure
		{"not found", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}, "HTTP_404"},
		// the response cannot be used, which is reported with the (successful) status of the response that had it
		{"malformed response", serve([]byte(`{"resource_metrics": "nope"}`)), "HTTP_200"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newTestServer(t, readFile(t, clusterInfoFile), test.handler)

			tierStatsState.reset()

			reporter, err := fetchOnce(t, server)

			require.NoError(t, err)

			errors := errorEvents(reporter.GetEvents())

			require.Len(t, errors, 1)
			require.Len(t, reporter.GetEvents(), 1)
			require.Equal(t, test.code, auto_ops_testing.GetObjectValue(errors[0].RootFields, "error.code"))
		})
	}
}

func TestFetchStateIsNotLostByAFailedScrape(t *testing.T) {
	var fail bool
	var mu sync.Mutex

	data := readFile(t, "./_meta/test/no_traffic.tier_stats.8.11.0.json")
	server := newTestServer(t, readFile(t, clusterInfoFile), func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if fail {
			w.WriteHeader(http.StatusServiceUnavailable)

			return
		}

		serve(data)(w, r)
	})

	ms := mbtest.NewReportingMetricSetV2Error(t, useNamedMetricSet(server.Server))

	tierStatsState.reset()

	require.NoError(t, ms.Fetch(&mbtest.CapturingReporterV2{}))

	mu.Lock()
	fail = true
	mu.Unlock()

	failed := &mbtest.CapturingReporterV2{}
	require.NoError(t, ms.Fetch(failed))
	require.Len(t, errorEvents(failed.GetEvents()), 1)

	mu.Lock()
	fail = false
	mu.Unlock()

	// the failure does not make the next scrape a first scrape, and the interval is measured from the last good scrape
	tierStatsState.mu.Lock()
	tierStatsState.previousTimestamp -= 20_000
	tierStatsState.mu.Unlock()

	recovered := &mbtest.CapturingReporterV2{}
	require.NoError(t, ms.Fetch(recovered))
	require.Len(t, recovered.GetEvents(), 2)

	for _, doc := range documents(recovered.GetEvents()) {
		require.GreaterOrEqual(t, doc["timestamp_diff"], int64(20_000))
	}
}

func TestFetchDoesNothingOnAClusterThatIsNotServerless(t *testing.T) {
	server := newTestServer(t, readFile(t, otherClusterInfoFile), func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("the metrics must not be requested from a cluster that is not serverless: %v %v", r.Method, r.RequestURI)
		w.WriteHeader(http.StatusInternalServerError)
	})

	tierStatsState.reset()

	for range 2 {
		reporter, err := fetchOnce(t, server)

		require.NoError(t, err)
		// not even an error event
		require.Empty(t, reporter.GetEvents())
		require.Empty(t, reporter.GetErrors())
	}

	for _, r := range server.recorded() {
		require.Equal(t, "/", r.uri)
	}
}
