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

package status

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/mongodb/mtest"
)

func TestStatus(t *testing.T) {
	mtest.Runner.Run(t, compose.Suite{
		"Fetch": testFetch,
		"Data":  testData,
	})
}

func testFetch(t *testing.T, r compose.R) {
	f := mbtest.NewReportingMetricSetV2(t, mtest.GetConfig("status", r.Host()))
	events, errs := mbtest.ReportingFetchV2(f)

	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
		events[0].BeatEvent("mongodb", "status").Fields.StringToPrint())

	event := events[0].BeatEvent("mongodb", "status").Fields

	// Check event fields
	current, _ := event.GetValue("mongodb.status.connections.current")
	assert.True(t, current.(int64) >= 0)

	available, _ := event.GetValue("mongodb.status.connections.available")
	assert.True(t, available.(int64) > 0)

	pageFaults, _ := event.GetValue("mongodb.status.extra_info.page_faults")
	assert.True(t, pageFaults.(int64) >= 0)
}

func testData(t *testing.T, r compose.R) {
	f := mbtest.NewReportingMetricSetV2(t, mtest.GetConfig("status", r.Host()))
	err := mbtest.WriteEventsReporterV2(f, t, ".")
	if err != nil {
		t.Fatal("write", err)
	}

}
