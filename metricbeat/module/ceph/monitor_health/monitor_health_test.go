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

package monitor_health

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
	absPath, err := filepath.Abs("../_meta/testdata/")
	assert.NoError(t, err)

	response, err := ioutil.ReadFile(absPath + "/sample_response.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "ceph",
		"metricsets": []string{"monitor_health"},
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

	mon := event
	assert.EqualValues(t, "HEALTH_OK", mon["health"])
	assert.EqualValues(t, "ceph", mon["name"])
	assert.EqualValues(t, "2017-01-19 11:34:50.700723 +0000 UTC", mon["last_updated"].(Tick).Time.String())

	available := mon["available"].(common.MapStr)
	assert.EqualValues(t, 4091244, available["kb"])
	assert.EqualValues(t, 65, available["pct"])

	total := mon["total"].(common.MapStr)
	assert.EqualValues(t, 6281216, total["kb"])

	used := mon["used"].(common.MapStr)
	assert.EqualValues(t, 2189972, used["kb"])

	store_stats := mon["store_stats"].(common.MapStr)
	assert.EqualValues(t, "0.000000", store_stats["last_updated"])

	misc := store_stats["misc"].(common.MapStr)
	assert.EqualValues(t, 840, misc["bytes"])

	log := store_stats["log"].(common.MapStr)
	assert.EqualValues(t, 8488103, log["bytes"])

	sst := store_stats["sst"].(common.MapStr)
	assert.EqualValues(t, 0, sst["bytes"])

	total = store_stats["total"].(common.MapStr)
	assert.EqualValues(t, 8488943, total["bytes"])
}
