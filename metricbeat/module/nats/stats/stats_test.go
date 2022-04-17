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

package stats

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
)

func TestEventMapping(t *testing.T) {
	content, err := ioutil.ReadFile("./_meta/test/statsmetrics.json")
	assert.NoError(t, err)
	reporter := &mbtest.CapturingReporterV2{}
	err = eventMapping(reporter, content)
	assert.NoError(t, err)
	event := reporter.GetEvents()[0]
	d, _ := event.MetricSetFields.GetValue("total_connections")
	assert.Equal(t, d, int64(35))
}

func TestFetchEventContent(t *testing.T) {
	absPath, _ := filepath.Abs("./_meta/test/")

	response, _ := ioutil.ReadFile(absPath + "/statsmetrics.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "nats",
		"metricsets": []string{"stats"},
		"hosts":      []string{server.URL},
	}
	reporter := &mbtest.CapturingReporterV2{}

	metricSet := mbtest.NewReportingMetricSetV2Error(t, config)
	metricSet.Fetch(reporter)

	e := mbtest.StandardizeEvent(metricSet, reporter.GetEvents()[0])
	t.Logf("%s/%s event: %+v", metricSet.Module().Name(), metricSet.Name(), e.Fields.StringToPrint())
}
