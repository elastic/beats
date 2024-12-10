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

package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/postgresql"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
)

func TestMetricSet_Fetch(t *testing.T) {
	service := compose.EnsureUp(t, "pgbouncer")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	events, errs := mbtest.ReportingFetchV2Error(f)
	require.Empty(t, errs, "Expected no errors during fetch")
	require.NotEmpty(t, events, "Expected to receive at least one event")
	event := events[0].MetricSetFields
	assert.Contains(t, event, "total_xact_count")
	assert.Contains(t, event, "total_server_assignment_count")
	assert.Contains(t, event, "total_wait_time_us")
	assert.Contains(t, event, "database")
	assert.Contains(t, event, "total_received")
	assert.Contains(t, event, "total_query_count")
}
func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "pgbouncer",
		"metricsets": []string{"stats"},
		"hosts":      []string{"localhost:6432/pgbouncer?sslmode=disable"},
		"username":   "test",
		"password":   postgresql.GetEnvPassword(),
	}
}
