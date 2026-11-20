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

//go:build darwin || freebsd || linux || openbsd || aix

package load

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	_ "github.com/elastic/beats/v7/metricbeat/module/system"
)

func TestFetch(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	events, errs := mbtest.ReportingFetchV2Error(f)

	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
		events[0].BeatEvent("system", "load").Fields.StringToPrint())
}

func TestFetchHostFS(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("test requires linux")
	}
	conf := getConfig()
	conf["hostfs"] = "./_meta/testdata"
	f := mbtest.NewReportingMetricSetV2Error(t, conf)
	t.Setenv("HOST_PROC", "/a/broken/path")
	events, errs := mbtest.ReportingFetchV2Error(f)
	assert.Empty(t, errs, "fetch must not fail")
	if !assert.Len(t, events, 1, "fetch must report one event") {
		return
	}

	fields := events[0].MetricSetFields
	assert.InDelta(t, 12.34, fields["1"], 1e-9, "1m load must come from the hostfs fixture")
	assert.InDelta(t, 5.67, fields["5"], 1e-9, "5m load must come from the hostfs fixture")
	assert.InDelta(t, 8.91, fields["15"], 1e-9, "15m load must come from the hostfs fixture")
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
		"module":     "system",
		"metricsets": []string{"load"},
	}
}
