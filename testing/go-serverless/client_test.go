// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package serverless

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient returns a Client talking to srv with a short poll interval.
func newTestClient(srv *httptest.Server) *Client {
	c := NewClient(srv.URL, "test-key", ProjectTypeObservability)
	c.pollInterval = 10 * time.Millisecond
	return c
}

func TestPickRegion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/serverless/regions", r.URL.Path, "unexpected path")
		assert.Equal(t, "ApiKey test-key", r.Header.Get("Authorization"), "missing API key")
		_, _ = w.Write([]byte(`[{"id":"aws-us-east-1","name":"N. Virginia"},{"id":"gcp-us-central1","name":"Iowa"}]`))
	}))
	defer srv.Close()
	c := newTestClient(srv)

	tests := []struct {
		name      string
		preferred string
		want      string
	}{
		{name: "preferred region is available", preferred: "gcp-us-central1", want: "gcp-us-central1"},
		{name: "falls back to the first region", preferred: "gcp-us-west2", want: "aws-us-east-1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := c.PickRegion(t.Context(), tc.preferred)
			require.NoError(t, err, "PickRegion failed")
			assert.Equal(t, tc.want, got, "unexpected region")
		})
	}
}

func TestCreateWaitDelete(t *testing.T) {
	var kibanaCalls atomic.Int32
	var deleted atomic.Bool

	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("POST /api/v1/serverless/projects/observability", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req), "decoding create request")
		assert.Equal(t, "test-project", req["name"], "unexpected project name")
		assert.Equal(t, "aws-us-east-1", req["region_id"], "unexpected region")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"abc","type":"observability","name":"test-project"}`))
	})
	mux.HandleFunc("POST /api/v1/serverless/projects/observability/abc/_reset-internal-credentials", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"username":"admin","password":"secret"}`))
	})
	mux.HandleFunc("GET /api/v1/serverless/projects/observability/abc", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{"id": "abc", "endpoints": map[string]string{
			"elasticsearch": srv.URL + "/es",
			"kibana":        srv.URL + "/kb",
		}}
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("GET /es", func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		assert.True(t, ok && user == "admin" && pass == "secret", "Elasticsearch must be queried with the reset credentials")
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /kb/api/status", func(w http.ResponseWriter, r *http.Request) {
		level := "degraded"
		// Report ready only on the second call to exercise polling.
		if kibanaCalls.Add(1) > 1 {
			level = "available"
		}
		_, _ = w.Write([]byte(`{"status":{"overall":{"level":"` + level + `"}}}`))
	})
	mux.HandleFunc("DELETE /api/v1/serverless/projects/observability/abc", func(w http.ResponseWriter, r *http.Request) {
		deleted.Store(true)
		w.WriteHeader(http.StatusOK)
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()
	c := newTestClient(srv)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	proj, err := c.Create(ctx, "test-project", "aws-us-east-1")
	require.NoError(t, err, "Create failed")
	assert.Equal(t, "abc", proj.ID, "unexpected project ID")
	assert.Equal(t, "admin", proj.Credentials.Username, "credentials must come from the reset call")
	assert.Equal(t, "secret", proj.Credentials.Password, "credentials must come from the reset call")

	proj, err = c.WaitReady(ctx, proj)
	require.NoError(t, err, "WaitReady failed")
	assert.Equal(t, srv.URL+"/es", proj.Endpoints.Elasticsearch, "unexpected Elasticsearch endpoint")
	assert.Equal(t, srv.URL+"/kb", proj.Endpoints.Kibana, "unexpected Kibana endpoint")
	assert.GreaterOrEqual(t, kibanaCalls.Load(), int32(2), "WaitReady must poll Kibana until it is available")

	require.NoError(t, c.Delete(ctx, proj), "Delete failed")
	assert.True(t, deleted.Load(), "the project must be deleted")
}

func TestCreateReportsAPIErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"nope"}`))
	}))
	defer srv.Close()

	_, err := newTestClient(srv).Create(t.Context(), "p", "r")
	require.Error(t, err, "Create must fail on a non-201 response")
	assert.Contains(t, err.Error(), "403", "the status code must be in the error")
	assert.Contains(t, err.Error(), "nope", "the response body must be in the error")
}

func TestWaitReadyHonorsContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"abc"}`)) // never publishes endpoints
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	_, err := newTestClient(srv).WaitReady(ctx, Project{ID: "abc"})
	require.ErrorIs(t, err, context.DeadlineExceeded, "WaitReady must give up when the context expires")
	assert.Contains(t, err.Error(), "waiting for endpoints of project abc", "the error must say what was awaited")
}
