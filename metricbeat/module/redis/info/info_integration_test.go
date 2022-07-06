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

package info

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "redis")

	ms := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	events, err := mbtest.ReportingFetchV2Error(ms)
	if err != nil {
		t.Fatal("fetch", err)
	}
	if len(events) == 0 {
		t.Fatal("no events")
	}
	event := events[0].MetricSetFields

	t.Logf("%s/%s event: %+v", ms.Module().Name(), ms.Name(), event)

	// Check fields
	assert.Equal(t, 10, len(event))
	server := event["server"].(mapstr.M)
	assert.Equal(t, "standalone", server["mode"])
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "redis")

	ms := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	err := mbtest.WriteEventsReporterV2Error(ms, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "redis",
		"metricsets": []string{"info"},
		"hosts":      []string{host},
	}
}
