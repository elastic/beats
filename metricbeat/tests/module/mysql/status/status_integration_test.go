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

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/mysql"

	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "mysql")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host(), false))
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 errors, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	event := events[0].MetricSetFields
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	// Check event fields
	connections := event["connections"].(int64)
	open := event["open"].(common.MapStr)
	openTables := open["tables"].(int64)
	openFiles := open["files"].(int64)
	openStreams := open["streams"].(int64)

	assert.True(t, connections > 0)
	assert.True(t, openTables > 0)
	assert.True(t, openFiles >= 0)
	assert.True(t, openStreams == 0)
}

func TestFetchRaw(t *testing.T) {
	service := compose.EnsureUp(t, "mysql")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host(), true))
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 errors, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	event := events[0].MetricSetFields
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	// Check event fields
	cachedThreads := event["threads"].(common.MapStr)["cached"].(int64)
	assert.True(t, cachedThreads >= 0)

	rawData := event["raw"].(common.MapStr)

	// Make sure field was removed from raw fields as in schema
	_, exists := rawData["Threads_cached"]
	assert.False(t, exists)

	// Check a raw field if it is available
	_, exists = rawData["Slow_launch_threads"]
	assert.True(t, exists)
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "mysql")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host(), false))

	err := mbtest.WriteEventsReporterV2Error(f, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig(host string, raw bool) map[string]interface{} {
	return map[string]interface{}{
		"module":     "mysql",
		"metricsets": []string{"status"},
		"hosts":      []string{mysql.GetMySQLEnvDSN(host)},
		"raw":        raw,
	}
}
