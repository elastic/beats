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

// +build !integration

package bucket

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestFetchEventContents(t *testing.T) {
	absPath, err := filepath.Abs("./testdata/")
	assert.NoError(t, err)

	// response is a raw response from a couchbase
	response, err := ioutil.ReadFile(absPath + "/sample_response.json")
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "couchbase",
		"metricsets": []string{"bucket"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	event := events[0].MetricSetFields

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	assert.EqualValues(t, "membase", event["type"])
	assert.EqualValues(t, "beer-sample", event["name"])

	data := event["data"].(common.MapStr)
	data_used := data["used"].(common.MapStr)
	assert.EqualValues(t, 12597731, data_used["bytes"])

	disk := event["disk"].(common.MapStr)
	assert.EqualValues(t, 0, disk["fetches"])

	disk_used := disk["used"].(common.MapStr)
	assert.EqualValues(t, 16369008, disk_used["bytes"])

	memory := event["memory"].(common.MapStr)
	memory_used := memory["used"].(common.MapStr)
	assert.EqualValues(t, 53962160, memory_used["bytes"])

	quota := event["quota"].(common.MapStr)
	quota_ram := quota["ram"].(common.MapStr)
	assert.EqualValues(t, 104857600, quota_ram["bytes"])

	quota_use := quota["use"].(common.MapStr)
	assert.EqualValues(t, 51.46232604980469, quota_use["pct"])

	assert.EqualValues(t, 7303, event["item_count"])
	assert.EqualValues(t, 0, event["ops_per_sec"])
}
