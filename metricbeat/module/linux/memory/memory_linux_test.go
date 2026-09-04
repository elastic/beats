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

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	_ "github.com/elastic/beats/v7/metricbeat/module/linux"
	"github.com/elastic/beats/v7/pkg/systemmetrics/metric/system/resolve"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestPercents(t *testing.T) {
	res := resolve.NewTestResolver("./_meta/testdata/")
	data := mapstr.M{}
	err := FetchLinuxMemStats(data, res)
	assert.NoError(t, err, "FetchLinuxMemStats")

	assertValue(t, data, "page_stats.kswapd_efficiency.pct", float64(1))
	assertValue(t, data, "page_stats.direct_efficiency.pct", float64(0.7143))
	assertValue(t, data, "swap.used.pct", float64(0))
}

func TestPagesFields(t *testing.T) {
	res := resolve.NewTestResolver("./_meta/testdata/")
	data := mapstr.M{}
	err := FetchLinuxMemStats(data, res)
	assert.NoError(t, err, "FetchLinuxMemStats")

	assertValue(t, data, "page_stats.pgfree.pages", uint64(2077939388))
	assertValue(t, data, "page_stats.pgscan_direct.pages", uint64(7))
	assertValue(t, data, "page_stats.pgsteal_direct.pages", uint64(5))
}

func assertValue(t *testing.T, data mapstr.M, key string, expected any) {
	t.Helper()
	actual, err := data.GetValue(key)
	if assert.NoError(t, err, "get %s", key) {
		assert.Equal(t, expected, actual, "value of %s", key)
	}
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

func getConfig() map[string]any {
	return map[string]any{
		"module":     "linux",
		"metricsets": []string{"memory"},
	}
}
