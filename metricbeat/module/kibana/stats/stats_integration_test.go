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

// +build integration

package stats

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/tests/compose"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/kibana"
	"github.com/elastic/beats/metricbeat/module/kibana/mtest"
)

func TestFetch(t *testing.T) {
	compose.EnsureUpWithTimeout(t, 600, "elasticsearch", "kibana")

	config := mtest.GetConfig("stats")
	host := config["hosts"].([]string)[0]
	version, err := getKibanaVersion(t, host)
	if err != nil {
		t.Fatal("getting kibana version", err)
	}

	isStatsAPIAvailable := kibana.IsStatsAPIAvailable(version)
	if err != nil {
		t.Fatal("checking if kibana stats API is available", err)
	}

	if !isStatsAPIAvailable {
		t.Skip("Kibana stats API is not available until 6.4.0")
	}

	f := mbtest.NewReportingMetricSetV2(t, config)
	events, errs := mbtest.ReportingFetchV2(f)

	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
		events[0].BeatEvent("kibana", "stats").Fields.StringToPrint())
}

func TestData(t *testing.T) {
	compose.EnsureUp(t, "kibana")

	config := mtest.GetConfig("stats")
	host := config["hosts"].([]string)[0]
	version, err := getKibanaVersion(t, host)
	if err != nil {
		t.Fatal("getting kibana version", err)
	}

	isStatsAPIAvailable := kibana.IsStatsAPIAvailable(version)
	if err != nil {
		t.Fatal("checking if kibana stats API is available", err)
	}

	if !isStatsAPIAvailable {
		t.Skip("Kibana stats API is not available until 6.4.0")
	}

	f := mbtest.NewReportingMetricSetV2(t, config)
	err = mbtest.WriteEventsReporterV2(f, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}

func getKibanaVersion(t *testing.T, kibanaHostPort string) (*common.Version, error) {
	resp, err := http.Get("http://" + kibanaHostPort + "/api/status")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data common.MapStr
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	version, err := data.GetValue("version.number")
	if err != nil {
		t.Log("Kibana GET /api/status response:", string(body))
		return nil, err
	}

	return common.NewVersion(version.(string))
}
