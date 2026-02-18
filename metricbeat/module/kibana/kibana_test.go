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

package kibana_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/module/kibana"
	"github.com/elastic/elastic-agent-libs/version"

	// Make sure metricsets are registered in mb.Registry
	_ "github.com/elastic/beats/v7/metricbeat/module/kibana/stats"
)

func TestIsStatsAPIAvailable(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"6.3.1", false},
		{"6.4.0", true},
		{"6.5.0", true},
		{"7.0.0-alpha1", true},
	}

	for _, test := range tests {
		actual := kibana.IsStatsAPIAvailable(version.MustNew(test.input))
		require.Equal(t, test.expected, actual)
	}
}

func TestReadBody(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{"200 returns body", 200, `{"status":"ok"}`, false},
		{"503 returns body", 503, `{"status":"degraded"}`, false},
		{"404 returns error", 404, "", true},
		{"500 returns error", 500, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}
			body, err := kibana.ReadBody(resp)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.body, string(body))
		})
	}
}
