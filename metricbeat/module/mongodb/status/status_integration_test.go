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

package status

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/tests/compose"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "mongodb")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	events, errs := mbtest.ReportingFetchV2Error(f)

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

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "mongodb")

	config := getConfig(service.Host())
	f := mbtest.NewReportingMetricSetV2Error(t, config)
	err := mbtest.WriteEventsReporterV2Error(f, t, ".")
	if err != nil {
		t.Fatal("write", err)
	}

}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "mongodb",
		"metricsets": []string{"status"},
		"hosts":      []string{host},
	}
}
