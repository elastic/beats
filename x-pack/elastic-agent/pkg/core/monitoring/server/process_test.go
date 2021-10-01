// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseID(t *testing.T) {
	cases := []struct {
		Name               string
		ID                 string
		ExpectedError      bool
		ExpectedStatusCode int
		ExpectedProgram    programDetail
	}{
		{"path injected id", ".././../etc/passwd", true, http.StatusBadRequest, programDetail{}},
		{"pipe injected id", "first | second", true, http.StatusBadRequest, programDetail{}},
		{"filebeat with suffix", "filebeat;cat demo-default-monitoring", true, http.StatusBadRequest, programDetail{}},

		{"filebeat correct", "filebeat-default", false, http.StatusBadRequest, programDetail{output: "default", binaryName: "filebeat"}},
		{"filebeat monitor correct", "filebeat-default-monitoring", false, http.StatusBadRequest, programDetail{output: "default", binaryName: "filebeat", isMonitoring: true}},

		{"mb correct", "metricbeat-default", false, http.StatusBadRequest, programDetail{output: "default", binaryName: "metricbeat"}},
		{"mb monitor correct", "metricbeat-default-monitoring", false, http.StatusBadRequest, programDetail{output: "default", binaryName: "metricbeat", isMonitoring: true}},

		{"endpoint correct", "endpoint-security-default", false, http.StatusBadRequest, programDetail{output: "default", binaryName: "endpoint-security"}},
		{"endpoint monitor correct", "endpoint-security-default-monitoring", false, http.StatusBadRequest, programDetail{output: "default", binaryName: "endpoint-security", isMonitoring: true}},

		{"unknown", "unknown-default", true, http.StatusNotFound, programDetail{}},
		{"unknown monitor", "unknown-default-monitoring", true, http.StatusNotFound, programDetail{}},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			pd, err := parseID(tc.ID)
			if !tc.ExpectedError {
				require.NoError(t, err)
			}

			if tc.ExpectedStatusCode > 0 && tc.ExpectedError {
				statErr, ok := err.(apiError)
				require.True(t, ok)
				require.Equal(t, tc.ExpectedStatusCode, statErr.Status())
			}

			require.Equal(t, tc.ExpectedProgram.binaryName, pd.binaryName)
			require.Equal(t, tc.ExpectedProgram.output, pd.output)
			require.Equal(t, tc.ExpectedProgram.isMonitoring, pd.isMonitoring)
		})
	}
}

func TestStatusErr(t *testing.T) {
	cases := map[string]struct {
		Error              error
		ExpectedStatusCode int
	}{
		"no error":                       {nil, 0},
		"normal error":                   {errors.New("something bad happened"), http.StatusInternalServerError},
		"status bound err - not found":   {errorWithStatus(http.StatusNotFound, errors.New("something was not found")), http.StatusNotFound},
		"status bound err - internal":    {errorWithStatus(http.StatusInternalServerError, errors.New("something was not found")), http.StatusInternalServerError},
		"status bound err - bad request": {errorWithStatus(http.StatusBadRequest, errors.New("something really bad happened")), http.StatusBadRequest},
	}

	dummyHandler := func(err error) func(w http.ResponseWriter, r *http.Request) error {
		return func(w http.ResponseWriter, r *http.Request) error {
			return err
		}
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			h := createHandler(dummyHandler(tc.Error))
			tw := &testWriter{}
			r, err := http.NewRequest("GET", "", nil)
			require.NoError(t, err)

			h.ServeHTTP(tw, r)

			require.Equal(t, tc.ExpectedStatusCode, tw.statusCode)
		})
	}
}

func TestProcessProxyRequest(t *testing.T) {
	sock := "/tmp/elastic-agent-test.sock"
	defer os.Remove(sock)

	endpoint := "http+unix://" + sock
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write the path to the client so they can verify the request
		// was correct
		w.Write([]byte(r.URL.Path))
	}))

	// Mimic subprocesses and listen on a unix socket
	l, err := net.Listen("unix", sock)
	require.NoError(t, err)
	server.Listener = l
	server.Start()
	defer server.Close()

	for _, path := range []string{"/stats", "/", "/state"} {
		respBytes, _, err := processMetrics(context.Background(), endpoint, path)
		require.NoError(t, err)
		// Verify that the server saw the path we tried to request
		assert.Equal(t, path, string(respBytes))
	}
}
