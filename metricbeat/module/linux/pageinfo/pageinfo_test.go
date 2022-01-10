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

package pageinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	_ "github.com/elastic/beats/v7/metricbeat/module/linux"
)

func TestData(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	err := mbtest.WriteEventsReporterV2Error(f, t, ".")
	if err != nil {
		t.Fatal("write", err)
	}
}

func TestFetch(t *testing.T) {
	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	events, errs := mbtest.ReportingFetchV2Error(f)

	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}

	testBuddyInfo := buddyInfo{
		DMA:    map[int]int64{0: 1, 1: 0, 2: 1, 3: 0, 4: 2, 5: 1, 6: 1, 7: 0, 8: 1, 9: 1, 10: 3},
		DMA32:  map[int]int64{0: 3, 1: 4, 2: 1, 3: 2, 4: 2, 5: 2, 6: 2, 7: 2, 8: 2, 9: 3, 10: 745},
		Normal: map[int]int64{0: 3409, 1: 4916, 2: 4010, 3: 4853, 4: 3613, 5: 2472, 6: 793, 7: 552, 8: 189, 9: 148, 10: 292},
	}

	rawEvent := events[0].BeatEvent("linux", "pageinfo").Fields["linux"].(common.MapStr)["pageinfo"].(common.MapStr)["buddy_info"]

	assert.Equal(t, testBuddyInfo, rawEvent)

}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "linux",
		"metricsets": []string{"pageinfo"},
		"hostfs":     "./_meta/testdata",
	}
}
