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

//go:build darwin || freebsd || linux || windows
// +build darwin freebsd linux windows

package process

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	events, errs := mbtest.ReportingFetchV2Error(f)

	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}

	// We have root cgroups disabled
	// This will pick a "populated" event to print
	if runtime.GOOS == "linux" {
		for _, evt := range events {
			field := evt.BeatEvent("system", "process").Fields["system"].(common.MapStr)["process"].(common.MapStr)["cgroup"].(common.MapStr)["cpu"]
			if field == nil {
				continue
			}
			if field.(map[string]interface{})["path"].(string) == "/" {
				continue
			}
			t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
				evt.BeatEvent("system", "process").Fields.StringToPrint())
			return
		}
	} else {
		t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
			events[0].BeatEvent("system", "process").Fields.StringToPrint())
	}

}

func TestData(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())

	// Do a first fetch to have percentages
	mbtest.ReportingFetchV2Error(f)
	time.Sleep(10 * time.Second)

	err := mbtest.WriteEventsReporterV2Error(f, t, ".")
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"process"},
		//"processes":  []string{".*metricbeat.*"}, // in case we want a prettier looking example for data.json
	}
}
