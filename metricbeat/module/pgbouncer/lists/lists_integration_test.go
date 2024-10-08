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

package lists

import (
	"fmt"
	"testing"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/postgresql"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
)

func TestMetricSet_Fetch(t *testing.T) {
	service := compose.EnsureUp(t, "pgbouncer")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	fmt.Printf("%v", f)
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	event := events[0].MetricSetFields
	assert.Contains(t, event, "databases")
	assert.Contains(t, event, "users")
	assert.Contains(t, event, "peers")
	assert.Contains(t, event, "pools")
	assert.Contains(t, event, "peer_pools")
	assert.Contains(t, event, "used_clients")
	assert.Contains(t, event, "free_servers")
	assert.Contains(t, event, "used_servers")
}
func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "pgbouncer",
		"metricsets": []string{"lists"},
		"hosts":      []string{"localhost:6432/pgbouncer?sslmode=disable"},
		"username":   "test",
		"password":   postgresql.GetEnvPassword(),
	}
}
