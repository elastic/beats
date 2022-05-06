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

//go:build integration
// +build integration

package stats

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/tests/compose"
	"github.com/elastic/elastic-agent-libs/mapstr"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/kibana"
	"github.com/elastic/beats/v7/metricbeat/module/kibana/mtest"
)

func TestFetch(t *testing.T) {
	service := compose.EnsureUpWithTimeout(t, 570, "kibana")

	config := mtest.GetConfig("stats", service.Host(), false)
	host := config["hosts"].([]string)[0]
	version, err := getKibanaVersion(t, host)
	require.NoError(t, err)

	isStatsAPIAvailable := kibana.IsStatsAPIAvailable(version)
	require.NoError(t, err)

	if !isStatsAPIAvailable {
		t.Skip("Kibana stats API is not available until 6.4.0")
	}

	f := mbtest.NewReportingMetricSetV2Error(t, config)
	events, errs := mbtest.ReportingFetchV2Error(f)

	require.Empty(t, errs)
	require.NotEmpty(t, events)

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
		events[0].BeatEvent("kibana", "stats").Fields.StringToPrint())
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "kibana")

	config := mtest.GetConfig("stats", service.Host(), false)
	host := config["hosts"].([]string)[0]
	version, err := getKibanaVersion(t, host)
	require.NoError(t, err)

	isStatsAPIAvailable := kibana.IsStatsAPIAvailable(version)
	require.NoError(t, err)

	if !isStatsAPIAvailable {
		t.Skip("Kibana stats API is not available until 6.4.0")
	}

	f := mbtest.NewReportingMetricSetV2Error(t, config)
	err = mbtest.WriteEventsReporterV2Error(f, t, "")
	require.NoError(t, err)
}

func getKibanaVersion(t *testing.T, kibanaHostPort string) (*common.Version, error) {
	resp, err := http.Get("http://" + kibanaHostPort + "/" + kibana.StatusPath)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data mapstr.M
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	version, err := data.GetValue("version.number")
	if err != nil {
		t.Log("Kibana GET /"+kibana.StatusPath+" response:", string(body))
		return nil, err
	}

	return common.NewVersion(version.(string))
}
