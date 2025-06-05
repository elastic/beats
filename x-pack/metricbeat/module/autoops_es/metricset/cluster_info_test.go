// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package metricset

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	autoopsevents "github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"

	"github.com/stretchr/testify/require"
)

func setupClusterInfoFromFile(filename string) auto_ops_testing.SetupServerCallback {
	return func(t *testing.T, _ []byte, _ []byte, _ string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.RequestURI {
			case "/":
				info, err := os.ReadFile(filename)
				require.NoError(t, err)

				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				w.Write(info)
			default:
				t.Fatalf("Unrecognized request %v", r.RequestURI)
			}
		}))
	}
}

func TestNestedFailedClusterInfoNoId(t *testing.T) {
	os.Setenv("DEPLOYMENT_ID", "test-resource-id")
	RunTestsForFetcherWithGlobFiles(t, "./_meta/test/success.7.17.0.json", setupClusterInfoFromFile("./_meta/test/no_id.cluster_info.7.17.0.json"), useNestedTestMetricSet, func(t *testing.T, data FetcherData[testObjectType]) {
		require.ErrorContains(t, data.Error, "failed to get cluster info from cluster, "+NESTED_NAME+" metricset")
		require.ErrorContains(t, data.Error, "cluster ID is unset, which means the cluster is not ready")

		// Check error event
		event := data.Reporter.GetEvents()[0]
		errorField, ok := event.MetricSetFields["error"].(autoopsevents.ErrorEvent)
		require.True(t, ok, "error field should be of type error.ErrorEvent")
		require.Equal(t, "CLUSTER_NOT_READY", errorField.ErrorCode)
		require.Contains(t, errorField.ErrorMessage, "failed to get cluster info from cluster, test_nested_metricset metricset")
		require.Equal(t, "", errorField.ClusterID)
		require.Equal(t, "/", errorField.URLPath)
		require.Equal(t, "test_nested_metricset", errorField.MetricSet)
		require.Equal(t, "test-resource-id", errorField.ResourceID)
		require.Equal(t, http.MethodGet, errorField.HTTPMethod)
		require.Equal(t, 0, errorField.HTTPStatusCode) // status code vary based on the server response for cluster not ready
	})
}

func TestNestedFailedClusterInfoNAId(t *testing.T) {
	os.Setenv("DEPLOYMENT_ID", "test-resource-id")
	RunTestsForFetcherWithGlobFiles(t, "./_meta/test/success.8.15.3.json", setupClusterInfoFromFile("./_meta/test/na_id.cluster_info.8.15.3.json"), useTestMetricSet, func(t *testing.T, data FetcherData[testObjectType]) {
		require.ErrorContains(t, data.Error, "failed to get cluster info from cluster, "+NAME+" metricset")
		require.ErrorContains(t, data.Error, "cluster ID is unset, which means the cluster is not ready")

		// Check error event
		event := data.Reporter.GetEvents()[0]
		errorField, ok := event.MetricSetFields["error"].(autoopsevents.ErrorEvent)
		require.True(t, ok, "error field should be of type error.ErrorEvent")
		require.Equal(t, "CLUSTER_NOT_READY", errorField.ErrorCode)
		require.Contains(t, errorField.ErrorMessage, "failed to get cluster info from cluster, test_metricset metricset")
		require.Equal(t, "", errorField.ClusterID)
		require.Equal(t, "/", errorField.URLPath)
		require.Equal(t, "test_metricset", errorField.MetricSet)
		require.Equal(t, "test-resource-id", errorField.ResourceID)
	})
}
