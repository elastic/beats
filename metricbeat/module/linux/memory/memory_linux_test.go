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

package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	_ "github.com/elastic/beats/v7/metricbeat/module/linux"
)

func TestPercents(t *testing.T) {
	res := resolve.NewTestResolver("./_meta/testdata/")
	data := common.MapStr{}
	err := FetchLinuxMemStats(data, res)
	assert.NoError(t, err, "FetchLinuxMemStats")

	assert.Equal(t, float64(1), data["page_stats"].(common.MapStr)["kswapd_efficiency"].(common.MapStr)["pct"].(float64))
	assert.Equal(t, float64(0.7143), data["page_stats"].(common.MapStr)["direct_efficiency"].(common.MapStr)["pct"].(float64))
}

func TestPagesFields(t *testing.T) {
	res := resolve.NewTestResolver("./_meta/testdata/")
	data := common.MapStr{}
	err := FetchLinuxMemStats(data, res)
	assert.NoError(t, err, "FetchLinuxMemStats")

	assert.Equal(t, uint64(2077939388), data["page_stats"].(common.MapStr)["pgfree"].(common.MapStr)["pages"].(uint64))
	assert.Equal(t, uint64(7), data["page_stats"].(common.MapStr)["pgscan_direct"].(common.MapStr)["pages"].(uint64))
	assert.Equal(t, uint64(5), data["page_stats"].(common.MapStr)["pgsteal_direct"].(common.MapStr)["pages"].(uint64))
}

func TestFetch(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	events, errs := mbtest.ReportingFetchV2Error(f)

	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
		events[0].BeatEvent("linux", "memory").Fields.StringToPrint())
}

func TestData(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	err := mbtest.WriteEventsReporterV2Error(f, t, ".")
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "linux",
		"metricsets": []string{"memory"},
	}
}
