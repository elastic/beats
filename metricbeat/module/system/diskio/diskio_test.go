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

//go:build integration && ((darwin && cgo) || freebsd || linux || windows)
// +build integration
// +build darwin,cgo freebsd linux windows

package diskio

import (
	"testing"
	"time"

	"github.com/menderesk/beats/v7/metricbeat/internal/sysinit"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestDataNameFilter(t *testing.T) {
	sysinit.InitModule("./_meta/testdata")
	conf := map[string]interface{}{
		"module":                 "system",
		"metricsets":             []string{"diskio"},
		"diskio.include_devices": []string{"sdb", "sdb1", "sdb2"},
		"hostfs":                 "./_meta/testdata",
	}

	f := mbtest.NewReportingMetricSetV2Error(t, conf)
	data, errs := mbtest.ReportingFetchV2Error(f)
	assert.Empty(t, errs)
	assert.Equal(t, 3, len(data))
}

func TestDataEmptyFilter(t *testing.T) {
	conf := map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"diskio"},
		"hostfs":     "./_meta/testdata",
	}

	f := mbtest.NewReportingMetricSetV2Error(t, conf)
	data, errs := mbtest.ReportingFetchV2Error(f)
	assert.Empty(t, errs)
	assert.Equal(t, 10, len(data))

}

func TestFetch(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	events, errs := mbtest.ReportingFetchV2Error(f)

	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
		events[0].BeatEvent("system", "diskio").Fields.StringToPrint())
}

func TestData(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	err := mbtest.WriteEventsReporterV2Error(f, t, ".")

	// Do a first fetch to have a sample
	mbtest.ReportingFetchV2Error(f)
	time.Sleep(1 * time.Second)

	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"diskio"},
	}
}
