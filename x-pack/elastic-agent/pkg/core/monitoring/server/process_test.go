// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
package server

import (
	"net/http"
	"testing"

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
