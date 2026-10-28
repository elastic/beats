// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package metricset

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	clusterInfo      = `{"cluster_name":"c","cluster_uuid":"abc","version":{"number":"8.11.0","build_flavor":"serverless"}}`
	otherClusterInfo = `{"cluster_name":"c","cluster_uuid":"abc","version":{"number":"8.15.3","build_flavor":"default"}}`
	dataPath         = "/_data?x=1"
	requestBody      = `{"ask":["a","b"]}`
)

type payload struct {
	Value int `json:"value"`
}

// received is a request that the server got.
type received struct {
	method      string
	uri         string
	contentType string
	body        string
}

type serverlessServer struct {
	*httptest.Server

	mu       sync.Mutex
	requests []received
}

// newServerlessServer serves the cluster info on `/` and `handler` on the data path.
func newServerlessServer(t *testing.T, clusterInfo string, handler http.HandlerFunc) *serverlessServer {
	t.Helper()

	s := &serverlessServer{}

	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))

		s.mu.Lock()
		s.requests = append(s.requests, received{method: r.Method, uri: r.RequestURI, contentType: r.Header.Get("Content-Type"), body: string(body)})
		s.mu.Unlock()

		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(clusterInfo))
		case "/_data":
			handler(w, r)
		default:
			assert.Failf(t, "unrecognized request", "%v %v", r.Method, r.RequestURI)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))

	t.Cleanup(s.Close)

	return s
}

func (s *serverlessServer) recorded() []received {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]received(nil), s.requests...)
}

func respondWith(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
}

// mapperCalls records what the mapper was given.
type mapperCalls struct {
	mu     sync.Mutex
	values []int
}

func (c *mapperCalls) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.values)
}

// newServerlessTestMetricSet creates the metricset through its own registry, so that it does not need to be registered globally.
func newServerlessTestMetricSet(t *testing.T, server *serverlessServer, options RequestOptions, mapper EventsMapper[payload]) mb.ReportingMetricSetV2Error {
	t.Helper()

	registry := mb.NewRegister()
	registry.MustAddMetricSet(MODULE_NAME, "serverless_test", func(base mb.BaseMetricSet) (mb.MetricSet, error) {
		return newServerlessAutoOpsMetricSet(base, dataPath, options, mapper)
	},
		mb.WithHostParser(elasticsearch.HostParser),
		mb.DefaultMetricSet(),
	)

	ms := mbtest.NewMetricSetWithRegistry(t, map[string]any{
		"module":     "autoops_es",
		"metricsets": []string{"serverless_test"},
		"hosts":      []string{server.URL},
	}, registry)

	reporting, ok := ms.(mb.ReportingMetricSetV2Error)
	require.True(t, ok)

	return reporting
}

// recordingMapper stores the values and reports an event for each one.
func recordingMapper(calls *mapperCalls) EventsMapper[payload] {
	return func(r mb.ReporterV2, info *utils.ClusterInfo, data *payload) error {
		calls.mu.Lock()
		calls.values = append(calls.values, data.Value)
		calls.mu.Unlock()

		r.Event(mb.Event{MetricSetFields: mapstr.M{"value": data.Value, "cluster": info.ClusterName}})

		return nil
	}
}

func isErrorEvent(event mb.Event) bool {
	_, isError := event.RootFields["error"]

	return isError
}

func options() RequestOptions {
	return RequestOptions{Method: http.MethodPost, Body: []byte(requestBody)}
}

func TestServerlessAutoOpsMetricSetSendsTheRequestAndOnlyThatRequest(t *testing.T) {
	server := newServerlessServer(t, clusterInfo, respondWith(`{"value": 7}`))
	calls := &mapperCalls{}
	ms := newServerlessTestMetricSet(t, server, options(), recordingMapper(calls))

	for range 2 {
		reporter := &mbtest.CapturingReporterV2{}

		require.NoError(t, ms.Fetch(reporter))
		require.Len(t, reporter.GetEvents(), 1)
		require.Equal(t, 7, reporter.GetEvents()[0].MetricSetFields["value"])
		require.Equal(t, "c", reporter.GetEvents()[0].MetricSetFields["cluster"])
	}

	requests := server.recorded()

	// every scrape asks for the cluster info, and then for the data
	require.Len(t, requests, 4)

	for i := 0; i < len(requests); i += 2 {
		// the request with a body does not change the request for the cluster info that follows it (the Content-Type header that it
		// sets stays, which is harmless without a body)
		require.Equal(t, http.MethodGet, requests[i].method)
		require.Equal(t, "/", requests[i].uri)
		require.Empty(t, requests[i].body)

		require.Equal(t, http.MethodPost, requests[i+1].method)
		require.Equal(t, dataPath, requests[i+1].uri)
		require.Equal(t, "application/json", requests[i+1].contentType)
		require.JSONEq(t, requestBody, requests[i+1].body)
	}

	require.Equal(t, 2, calls.count())
}

func TestServerlessAutoOpsMetricSetUsesTheConfiguredMethod(t *testing.T) {
	server := newServerlessServer(t, clusterInfo, respondWith(`{"value": 1}`))
	ms := newServerlessTestMetricSet(t, server, RequestOptions{Method: http.MethodGet, Body: []byte(requestBody)}, recordingMapper(&mapperCalls{}))

	require.NoError(t, ms.Fetch(&mbtest.CapturingReporterV2{}))

	requests := server.recorded()

	require.Len(t, requests, 2)
	require.Equal(t, http.MethodGet, requests[1].method)
	require.JSONEq(t, requestBody, requests[1].body)
}

func TestServerlessAutoOpsMetricSetReportsOtherFailures(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"server error", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }},
		{"too many requests", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTooManyRequests) }},
		// there is no special handling of the route not being there (yet), such as while the API is being rolled out
		{"not found", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) }},
		{"malformed response", respondWith(`{"value": "not a number"}`)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newServerlessServer(t, clusterInfo, test.handler)
			calls := &mapperCalls{}
			ms := newServerlessTestMetricSet(t, server, options(), recordingMapper(calls))

			reporter := &mbtest.CapturingReporterV2{}

			require.NoError(t, ms.Fetch(reporter))
			require.Len(t, reporter.GetEvents(), 1)
			require.True(t, isErrorEvent(reporter.GetEvents()[0]))
			require.Zero(t, calls.count())
		})
	}
}

func TestServerlessAutoOpsMetricSetReportsAClusterInfoFailure(t *testing.T) {
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(failing.Close)

	calls := &mapperCalls{}
	ms := newServerlessTestMetricSet(t, &serverlessServer{Server: failing}, options(), recordingMapper(calls))

	reporter := &mbtest.CapturingReporterV2{}

	require.NoError(t, ms.Fetch(reporter))
	require.Len(t, reporter.GetEvents(), 1)
	require.True(t, isErrorEvent(reporter.GetEvents()[0]))
	require.Zero(t, calls.count())
}

func TestServerlessAutoOpsMetricSetMapperErrorsAreTheMappersToReport(t *testing.T) {
	server := newServerlessServer(t, clusterInfo, respondWith(`{"value": 9}`))
	ms := newServerlessTestMetricSet(t, server, options(), func(_ mb.ReporterV2, _ *utils.ClusterInfo, _ *payload) error {
		return errors.New("the mapper failed")
	})

	reporter := &mbtest.CapturingReporterV2{}

	// like AutoOpsMetricSet, Fetch does not return the error, nor report it a second time
	require.NoError(t, ms.Fetch(reporter))
	require.Empty(t, reporter.GetEvents())
	require.Empty(t, reporter.GetErrors())
}

func TestServerlessAutoOpsMetricSetRecoversAfterAFailure(t *testing.T) {
	var fail bool
	var mu sync.Mutex

	server := newServerlessServer(t, clusterInfo, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if fail {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		respondWith(`{"value": 4}`)(w, r)
	})
	calls := &mapperCalls{}
	ms := newServerlessTestMetricSet(t, server, options(), recordingMapper(calls))

	mu.Lock()
	fail = true
	mu.Unlock()

	// every scrape that fails is reported, because nothing is remembered about the failure
	for range 2 {
		reporter := &mbtest.CapturingReporterV2{}

		require.NoError(t, ms.Fetch(reporter))
		require.Len(t, reporter.GetEvents(), 1)
		require.True(t, isErrorEvent(reporter.GetEvents()[0]))
	}

	mu.Lock()
	fail = false
	mu.Unlock()

	reporter := &mbtest.CapturingReporterV2{}

	require.NoError(t, ms.Fetch(reporter))
	require.Len(t, reporter.GetEvents(), 1)
	require.False(t, isErrorEvent(reporter.GetEvents()[0]))
	require.Equal(t, 1, calls.count())
}

func TestServerlessAutoOpsMetricSetDoesNothingOnOtherClusters(t *testing.T) {
	server := newServerlessServer(t, otherClusterInfo, func(w http.ResponseWriter, r *http.Request) {
		assert.Fail(t, "the data must not be requested from a cluster that is not serverless", "%v %v", r.Method, r.RequestURI)
		w.WriteHeader(http.StatusInternalServerError)
	})
	calls := &mapperCalls{}
	ms := newServerlessTestMetricSet(t, server, options(), recordingMapper(calls))

	for range 3 {
		reporter := &mbtest.CapturingReporterV2{}

		require.NoError(t, ms.Fetch(reporter))
		// not even an error event
		require.Empty(t, reporter.GetEvents())
		require.Empty(t, reporter.GetErrors())
	}

	require.Zero(t, calls.count())

	// only the cluster info is requested
	requests := server.recorded()

	require.Len(t, requests, 3)

	for _, request := range requests {
		require.Equal(t, "/", request.uri)
	}
}

func TestServerlessAutoOpsMetricSetDoesNotUseCloudConnectedMode(t *testing.T) {
	previous := checkedCloudConnectedMode
	checkedCloudConnectedMode = false

	t.Cleanup(func() { checkedCloudConnectedMode = previous })
	t.Setenv(CLOUD_CONNECTED_MODE_API_KEY_NAME, "an-api-key")

	// any request other than the cluster info and the data fails the test, such as the ones that registering would make
	server := newServerlessServer(t, clusterInfo, respondWith(`{"value": 1}`))
	newServerlessTestMetricSet(t, server, options(), recordingMapper(&mapperCalls{}))

	require.False(t, checkedCloudConnectedMode, "creating the metricset must not check Cloud Connected mode")
	require.Empty(t, server.recorded(), "creating the metricset must not make any request")
}

func TestServerlessAutoOpsMetricSetSetsTheResourceID(t *testing.T) {
	utils.ClearResourceID()
	t.Cleanup(utils.ClearResourceID)
	t.Setenv("AUTOOPS_PROJECT_ID", "my-project")

	server := newServerlessServer(t, clusterInfo, respondWith(`{"value": 1}`))
	newServerlessTestMetricSet(t, server, options(), recordingMapper(&mapperCalls{}))

	require.Equal(t, "my-project", utils.GetResourceID(), "the resource ID must come from the environment")
}
