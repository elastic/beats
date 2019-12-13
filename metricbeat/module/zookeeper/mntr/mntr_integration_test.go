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

package mntr

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "zookeeper")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	events, errs := mbtest.ReportingFetchV2Error(f)

	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
		events[0].BeatEvent("zookeeper", "mntr").Fields.StringToPrint())

	e, _ := events[0].BeatEvent("zookeeper", "mntr").Fields.GetValue("zookeeper.mntr")
	event := e.(common.MapStr)
	// Check values
	avgLatency := event["latency"].(common.MapStr)["avg"].(int64)
	maxLatency := event["latency"].(common.MapStr)["max"].(int64)
	numAliveConnections := event["num_alive_connections"].(int64)

	assert.True(t, avgLatency >= 0)
	assert.True(t, maxLatency >= 0)
	assert.True(t, numAliveConnections > 0)

	// Check number of fields. At least 10, depending on environment
	assert.True(t, len(event) >= 10)
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "zookeeper")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	err := mbtest.WriteEventsReporterV2Error(f, t, ".")
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "zookeeper",
		"metricsets": []string{"mntr"},
		"hosts":      []string{host},
	}
}
