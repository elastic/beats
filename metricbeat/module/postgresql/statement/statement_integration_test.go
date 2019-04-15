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

package statement

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/postgresql"

	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "postgresql")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	event := events[0].MetricSetFields

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	// Check event fields
	assert.Contains(t, event, "user")
	assert.Contains(t, event["user"].(common.MapStr), "id")

	assert.Contains(t, event, "database")
	db_oid := event["database"].(common.MapStr)["oid"].(int64)
	assert.True(t, db_oid > 0)

	assert.Contains(t, event, "query")
	query := event["query"].(common.MapStr)
	assert.Contains(t, query, "id")
	assert.Contains(t, query, "text")
	assert.Contains(t, query, "calls")
	assert.Contains(t, query, "rows")

	assert.Contains(t, query, "time")
	time := query["time"].(common.MapStr)
	assert.Contains(t, time, "total")
	assert.Contains(t, time, "min")
	assert.Contains(t, time, "max")
	assert.Contains(t, time, "mean")
	assert.Contains(t, time, "stddev")

	assert.Contains(t, query["memory"], "shared")
	memory := query["memory"].(common.MapStr)

	assert.Contains(t, memory, "shared")
	shared := memory["shared"].(common.MapStr)
	assert.Contains(t, shared, "hit")
	assert.Contains(t, shared, "read")
	assert.Contains(t, shared, "dirtied")
	assert.Contains(t, shared, "written")

	assert.Contains(t, memory, "local")
	local := memory["local"].(common.MapStr)
	assert.Contains(t, local, "hit")
	assert.Contains(t, local, "read")
	assert.Contains(t, local, "dirtied")
	assert.Contains(t, local, "written")

	assert.Contains(t, memory, "temp")
	temp := memory["temp"].(common.MapStr)
	assert.Contains(t, temp, "read")
	assert.Contains(t, temp, "written")
}

func TestData(t *testing.T) {
	compose.EnsureUp(t, "postgresql")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig())
	if err := mbtest.WriteEventsReporterV2Error(f, t, ""); err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "postgresql",
		"metricsets": []string{"statement"},
		"hosts":      []string{postgresql.GetEnvDSN()},
		"username":   postgresql.GetEnvUsername(),
		"password":   postgresql.GetEnvPassword(),
	}
}
