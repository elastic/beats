// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cat_shards

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	autoopsevents "github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
)

var (
	setupSuccessfulServer = SetupSuccessfulServer()
	useNamedMetricSet     = auto_ops_testing.UseNamedMetricSet(CatShardsMetricSet)
)

func setupResolveElasticSearchServer(t *testing.T, clusterInfo []byte, data []byte, _ string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.RequestURI {
		case "/":
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			w.Write(clusterInfo)
		case CatShardsPath:
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
		case ResolveIndexPath:
			w.WriteHeader(500)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
					"error": {
						"root_cause": [
							{
								"type": "error_type",
								"reason": "error_reason"
							}
						],
						"type": "error_type",
						"reason": "error_reason"
					},
					"status": 500
				}`))
		default:
			t.Fatalf("Unknown request to %v", r.RequestURI)
		}
	}))
}

func setupResolve404ElasticSearchError(t *testing.T, clusterInfo []byte, data []byte, _ string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.RequestURI {
		case "/":
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			w.Write(clusterInfo)
		case CatShardsPath:
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
		case ResolveIndexPath:
			w.WriteHeader(404)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				  "error": {
					"root_cause": [
					  {
						"type": "index_not_found_exception",
						"reason": "no such index [test-index]",
						"resource.type": "index_or_alias",
						"resource.id": "test-index",
						"index_uuid": "_na_",
						"index": "test-index"
					  }
					],
					"type": "index_not_found_exception",
					"reason": "no such index [test-index]",
					"resource.type": "index_or_alias",
					"resource.id": "test-index",
					"index_uuid": "_na_",
					"index": "test-index"
				  },
				  "status": 404
				}`))
		default:
			t.Fatalf("Unknown request to %v", r.RequestURI)
		}
	}))
}

func setupResolve405ElasticSearchError(t *testing.T, clusterInfo []byte, data []byte, _ string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.RequestURI {
		case "/":
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			w.Write(clusterInfo)
		case CatShardsPath:
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
		case ResolveIndexPath:
			w.WriteHeader(405)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`
				{
				  "error": "Incorrect HTTP method for uri [/_cat/template?pretty=true] and method [GET], allowed: [POST]",
				  "status": 405
				}
			`))
		default:
			t.Fatalf("Unknown request to %v", r.RequestURI)
		}
	}))
}

func setupResolveEmptyErrorServer(t *testing.T, clusterInfo []byte, data []byte, _ string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.RequestURI {
		case "/":
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			w.Write(clusterInfo)
		case CatShardsPath:
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
		case ResolveIndexPath:
			w.WriteHeader(500)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(``))
		default:
			t.Fatalf("Unknown request to %v", r.RequestURI)
		}
	}))
}

func TestSuccessfulFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_shards.*.json", setupSuccessfulServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[[]JSONShard]) {
		require.NoError(t, data.Error)

		require.LessOrEqual(t, 1, len(data.Reporter.GetEvents()))
	})
}

func TestFailedClusterInfoFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_shards.*.json", auto_ops_testing.SetupClusterInfoErrorServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[[]JSONShard]) {
		require.ErrorContains(t, data.Error, "failed to get cluster info from cluster, cat_shards metricset")
	})
}

func TestFailedCatShardsFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_shards.*.json", auto_ops_testing.SetupDataErrorServer(CatShardsPath), useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[[]JSONShard]) {
		require.ErrorContains(t, data.Error, "failed to get data, cat_shards metricset")
	})
}

// Integration tests to check the error handling when the server returns different error types
// TestElasticSearchError tests the error handling when an Elasticsearch simple error is returned
func Test500FailedToResolveIndexesWhileFetching(t *testing.T) {
	os.Setenv("DEPLOYMENT_ID", "test-resource-id")
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_shards.*.json", setupResolveErrorServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[[]JSONShard]) {
		require.Error(t, data.Error)

		// Check error event
		event := data.Reporter.GetEvents()[1]
		errorField, ok := event.MetricSetFields["error"].(autoopsevents.ErrorEvent)
		require.True(t, ok, "error field should be of type *error.ErrorEvent")
		require.Equal(t, "HTTP_500", errorField.ErrorCode)
		require.Equal(t, "failed to load resolved index details failed to fetch data: HTTP error 500 Internal Server Error", errorField.ErrorMessage)
		require.Equal(t, "GZbSUUMQQI-A7UcGS6vCMa", errorField.ClusterID)
		require.Equal(t, "/_cat/shards", errorField.URLPath)
		require.Equal(t, "s=i&h=n,i,id,s,p,st,d,sto,sc,sqto,sqti,iito,iiti,iif,mt,mtt,gmto,gmti,ur,ud&bytes=b&time=ms&format=json", errorField.Query)
		require.Equal(t, "cat_shards", errorField.MetricSet)
		require.Equal(t, "test-resource-id", errorField.ResourceID)
		require.Equal(t, "GET", errorField.HTTPMethod)
		require.Equal(t, 500, errorField.HTTPStatusCode)
		require.Equal(t, "Server Error", errorField.HTTPResponse) // checking the HTTP response body
	})
}

// TestElasticSearchError tests the error handling when an elasticsearch is returned
func Test404FailedToResolveIndexesWhileFetching(t *testing.T) {
	os.Setenv("DEPLOYMENT_ID", "test-resource-id")
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_shards.*.json", setupResolve404ElasticSearchError, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[[]JSONShard]) {
		require.Error(t, data.Error)

		// Check error event
		event := data.Reporter.GetEvents()[1]
		errorField, ok := event.MetricSetFields["error"].(autoopsevents.ErrorEvent)
		require.True(t, ok, "error field should be of type error.ErrorEvent")
		require.Equal(t, "HTTP_404", errorField.ErrorCode)
		require.Equal(t, "failed to load resolved index details failed to fetch data: HTTP error 404 Not Found", errorField.ErrorMessage)
		require.Equal(t, "GZbSUUMQQI-A7UcGS6vCMa", errorField.ClusterID)
		require.Equal(t, "/_cat/shards", errorField.URLPath)
		require.Equal(t, "s=i&h=n,i,id,s,p,st,d,sto,sc,sqto,sqti,iito,iiti,iif,mt,mtt,gmto,gmti,ur,ud&bytes=b&time=ms&format=json", errorField.Query)
		require.Equal(t, "cat_shards", errorField.MetricSet)
		require.Equal(t, "test-resource-id", errorField.ResourceID)
		require.Equal(t, "GET", errorField.HTTPMethod)
		require.Equal(t, 404, errorField.HTTPStatusCode)
		// avoiding the HTTP response body check on purpose, as the error response is a JSON string, and it's already tested
	})
}

// TestElasticSearchError tests the error handling when an elasticsearch is returned
func Test405FailedToResolveIndexesWhileFetching(t *testing.T) {
	os.Setenv("DEPLOYMENT_ID", "test-resource-id")
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_shards.*.json", setupResolve405ElasticSearchError, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[[]JSONShard]) {
		require.Error(t, data.Error)

		// Check error event
		event := data.Reporter.GetEvents()[1]
		errorField, ok := event.MetricSetFields["error"].(autoopsevents.ErrorEvent)
		require.True(t, ok, "error field should be of type error.ErrorEvent")
		require.Equal(t, "HTTP_405", errorField.ErrorCode)
		require.Equal(t, "failed to load resolved index details failed to fetch data: HTTP error 405 Method Not Allowed", errorField.ErrorMessage)
		require.Equal(t, "test-resource-id", errorField.ResourceID)
		require.Equal(t, "GZbSUUMQQI-A7UcGS6vCMa", errorField.ClusterID)
		require.Equal(t, "/_cat/shards", errorField.URLPath)
		require.Equal(t, "s=i&h=n,i,id,s,p,st,d,sto,sc,sqto,sqti,iito,iiti,iif,mt,mtt,gmto,gmti,ur,ud&bytes=b&time=ms&format=json", errorField.Query)
		require.Equal(t, "cat_shards", errorField.MetricSet)
		require.Equal(t, "GET", errorField.HTTPMethod)
		require.Equal(t, 405, errorField.HTTPStatusCode)
		// avoiding the HTTP response body check on purpose, as the error response is a JSON string, and it's already tested
	})
}

// TestElasticSearchError tests the error handling when an error different from Elasticsearch is returned (proxy error, etc.)
func Test500FailedToResolveIndexesWhileFetchingEmptyResponse(t *testing.T) {
	os.Setenv("DEPLOYMENT_ID", "test-resource-id")
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_shards.*.json", setupResolveEmptyErrorServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[[]JSONShard]) {
		require.Error(t, data.Error)

		// Check error event
		event := data.Reporter.GetEvents()[1]
		errorField, ok := event.MetricSetFields["error"].(autoopsevents.ErrorEvent)
		require.True(t, ok, "error field should be of type error.ErrorEvent")
		require.Equal(t, "HTTP_500", errorField.ErrorCode)
		require.Equal(t, "failed to load resolved index details failed to fetch data: HTTP error 500 Internal Server Error", errorField.ErrorMessage)
		require.Equal(t, "test-resource-id", errorField.ResourceID)
		require.Equal(t, "GZbSUUMQQI-A7UcGS6vCMa", errorField.ClusterID)
		require.Equal(t, "/_cat/shards", errorField.URLPath)
		require.Equal(t, "s=i&h=n,i,id,s,p,st,d,sto,sc,sqto,sqti,iito,iiti,iif,mt,mtt,gmto,gmti,ur,ud&bytes=b&time=ms&format=json", errorField.Query)
		require.Equal(t, "cat_shards", errorField.MetricSet)
		require.Equal(t, "GET", errorField.HTTPMethod)
		require.Equal(t, 500, errorField.HTTPStatusCode)
		// avoiding the HTTP response body check on purpose, as the error response is a JSON string, and it's already tested
	})
}
