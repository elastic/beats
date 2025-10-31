// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package node_stats

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"

	"github.com/stretchr/testify/require"
)

func setupSuccessfulServer() auto_ops_testing.SetupServerCallback {
	return func(t *testing.T, clusterInfo []byte, data []byte, version string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.RequestURI {
			case "/":
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				w.Write(clusterInfo)
			case NodesStatsPath:
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				w.Write(data)
			case ClusterStateMasterNodePath:
				resolvedIndexes, err := os.ReadFile("./_meta/test/master_node." + version + ".json")
				require.NoError(t, err)

				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				w.Write(resolvedIndexes)
			default:
				t.Fatalf("Unknown request to %v", r.RequestURI)
			}
		}))
	}
}

func setupMasterNodeErrorServer(t *testing.T, clusterInfo []byte, data []byte, _ string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.RequestURI {
		case "/":
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			w.Write(clusterInfo)
		case NodesStatsPath:
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
		case ClusterStateMasterNodePath:
			w.WriteHeader(500)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"error":"Unexpected error"}`))
		default:
			t.Fatalf("Unknown request to %v", r.RequestURI)
		}
	}))
}
