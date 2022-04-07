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

package cluster_status

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/v8/libbeat/common"
	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestFetchEventContents(t *testing.T) {
	absPath, err := filepath.Abs("../_meta/testdata/")
	assert.NoError(t, err)

	response, err := ioutil.ReadFile(absPath + "/status_sample_response.json")
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "ceph",
		"metricsets": []string{"cluster_status"},
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

	//check status version number
	assert.EqualValues(t, 813, event["version"])

	//check osd info
	osdmap := event["osd"].(common.MapStr)
	assert.EqualValues(t, false, osdmap["full"])
	assert.EqualValues(t, false, osdmap["nearfull"])
	assert.EqualValues(t, 6, osdmap["osd_count"])
	assert.EqualValues(t, 3, osdmap["up_osd_count"])
	assert.EqualValues(t, 4, osdmap["in_osd_count"])
	assert.EqualValues(t, 240, osdmap["remapped_pg_count"])
	assert.EqualValues(t, 264, osdmap["epoch"])

	//check traffic info
	trafficInfo := event["traffic"].(common.MapStr)
	assert.EqualValues(t, 55667788, trafficInfo["read_bytes"])
	assert.EqualValues(t, 1234, trafficInfo["read_op_per_sec"])
	assert.EqualValues(t, 11996158, trafficInfo["write_bytes"])
	assert.EqualValues(t, 10, trafficInfo["write_op_per_sec"])

	//check misplace info
	misplaceInfo := event["misplace"].(common.MapStr)
	assert.EqualValues(t, 768, misplaceInfo["total"])
	assert.EqualValues(t, 88, misplaceInfo["objects"])
	assert.EqualValues(t, 0.114583, misplaceInfo["pct"])

	//check degraded info
	degradedInfo := event["degraded"].(common.MapStr)
	assert.EqualValues(t, 768, degradedInfo["total"])
	assert.EqualValues(t, 294, degradedInfo["objects"])
	assert.EqualValues(t, 0.382812, degradedInfo["pct"])

	//check pg info
	pgInfo := event["pg"].(common.MapStr)
	assert.EqualValues(t, 1054023794, pgInfo["data_bytes"])
	assert.EqualValues(t, int64(9965821952), pgInfo["avail_bytes"])
	assert.EqualValues(t, int64(12838682624), pgInfo["total_bytes"])
	assert.EqualValues(t, int64(2872860672), pgInfo["used_bytes"])

	//check pg_state info
	pgStateInfo := events[1].MetricSetFields["pg_state"].(common.MapStr)
	assert.EqualValues(t, "active+undersized+degraded", pgStateInfo["state_name"])
	assert.EqualValues(t, 109, pgStateInfo["count"])
	assert.EqualValues(t, 813, pgStateInfo["version"])

	pgStateInfo = events[2].MetricSetFields["pg_state"].(common.MapStr)
	assert.EqualValues(t, "undersized+degraded+peered", pgStateInfo["state_name"])
	assert.EqualValues(t, 101, pgStateInfo["count"])
	assert.EqualValues(t, 813, pgStateInfo["version"])

	pgStateInfo = events[3].MetricSetFields["pg_state"].(common.MapStr)
	assert.EqualValues(t, "active+remapped", pgStateInfo["state_name"])
	assert.EqualValues(t, 55, pgStateInfo["count"])
	assert.EqualValues(t, 813, pgStateInfo["version"])

	pgStateInfo = events[4].MetricSetFields["pg_state"].(common.MapStr)
	assert.EqualValues(t, "active+undersized+degraded+remapped", pgStateInfo["state_name"])
	assert.EqualValues(t, 55, pgStateInfo["count"])
	assert.EqualValues(t, 813, pgStateInfo["version"])
}
