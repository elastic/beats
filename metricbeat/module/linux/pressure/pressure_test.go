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

//go:build linux
// +build linux

package pressure

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/common"

	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
	_ "github.com/menderesk/beats/v7/metricbeat/module/linux"
)

func TestFetch(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	events, errs := mbtest.ReportingFetchV2Error(f)

	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
		events[0].BeatEvent("linux", "pressure").Fields.StringToPrint())

	resources := []string{"cpu", "memory", "io"}

	for i := range events {
		resource := resources[i]

		testEvent := common.MapStr{
			resource: common.MapStr{
				"some": common.MapStr{
					"10": common.MapStr{
						"pct": 5.86,
					},
					"60": common.MapStr{
						"pct": 1.10,
					},
					"300": common.MapStr{
						"pct": 0.23,
					},
					"total": common.MapStr{
						"time": common.MapStr{
							"us": uint64(9895236),
						},
					},
				},
			},
		}
		// /proc/pressure/cpu does not contain 'full' metrics
		if resource != "cpu" {
			testEvent.Put(resource+".full.10.pct", 6.86)
			testEvent.Put(resource+".full.60.pct", 2.10)
			testEvent.Put(resource+".full.300.pct", 1.23)
			testEvent.Put(resource+".full.total.time.us", uint64(10895236))
		}

		rawEvent := events[i].BeatEvent("linux", "pressure").Fields["linux"].(common.MapStr)["pressure"]
		assert.Equal(t, testEvent, rawEvent)
	}
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
		"metricsets": []string{"pressure"},
		"hostfs":     "./_meta/testdata",
	}
}
