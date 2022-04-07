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

package mgr_cluster_health

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common"
	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"
)

type clientRequest struct {
	Prefix string `json:"prefix"`
}

func TestFetchEventContents(t *testing.T) {
	absPath, err := filepath.Abs("./_meta/testdata/")
	assert.NoError(t, err)

	statusResponse, err := ioutil.ReadFile(absPath + "/status.json")
	assert.NoError(t, err)
	timeSyncStatusResponse, err := ioutil.ReadFile(absPath + "/time_sync_status.json")
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")

		defer r.Body.Close()
		var request clientRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		require.NoError(t, err)

		if request.Prefix == "status" {
			w.Write(statusResponse)
		} else if request.Prefix == "time-sync-status" {
			w.Write(timeSyncStatusResponse)
		}
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "ceph",
		"metricsets": []string{"mgr_cluster_health"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	event := events[0].ModuleFields["cluster_health"].(common.MapStr)

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	assert.EqualValues(t, "HEALTH_OK", event["overall_status"])

	timechecks := event["timechecks"].(common.MapStr)
	assert.EqualValues(t, 3, timechecks["epoch"])

	round := timechecks["round"].(common.MapStr)
	assert.EqualValues(t, 0, round["value"])
	assert.EqualValues(t, "finished", round["status"])
}

func TestFetchEventContents_Failed(t *testing.T) {
	absPath, err := filepath.Abs("./_meta/testdata/")
	assert.NoError(t, err)

	response, err := ioutil.ReadFile(absPath + "/failed.json")
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json;")
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := map[string]interface{}{
		"module":     "ceph",
		"metricsets": []string{"mgr_cluster_health"},
		"hosts":      []string{server.URL},
	}

	f := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(f)
	assert.Empty(t, events)
	assert.NotEmpty(t, errs)
}
