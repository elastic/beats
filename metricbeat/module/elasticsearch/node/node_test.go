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

//go:build !integration
// +build !integration

package node

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"
	"github.com/elastic/beats/v8/metricbeat/module/elasticsearch"
)

func TestFetch(t *testing.T) {

	files, err := filepath.Glob("./_meta/test/node.*.json")
	require.NoError(t, err)
	// Makes sure glob matches at least 1 file
	require.True(t, len(files) > 0)

	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			response, err := ioutil.ReadFile(f)
			require.NoError(t, err)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.RequestURI {
				case "/_nodes/_local":
					w.WriteHeader(200)
					w.Header().Set("Content-Type", "application/json;")
					w.Write([]byte(response))

				case "/":
					rootResponse := "{\"cluster_name\":\"es1\",\"cluster_uuid\":\"4heb1eiady103dxu71\",\"version\":{\"number\":\"7.0.0\"}}"
					w.WriteHeader(200)
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(rootResponse))

				default:
					t.FailNow()
				}

			}))
			defer server.Close()

			config := map[string]interface{}{
				"module":     elasticsearch.ModuleName,
				"metricsets": []string{"node"},
				"hosts":      []string{server.URL},
			}
			reporter := &mbtest.CapturingReporterV2{}

			metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
			metricSet.Fetch(reporter)

			e := mbtest.StandardizeEvent(metricSet, reporter.GetEvents()[0])
			t.Logf("%s/%s event: %+v", metricSet.Module().Name(), metricSet.Name(), e.Fields.StringToPrint())
		})
	}
}
