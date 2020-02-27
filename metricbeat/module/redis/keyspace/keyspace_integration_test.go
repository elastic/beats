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

package keyspace

import (
	"strings"
	"testing"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	rd "github.com/garyburd/redigo/redis"
	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "redis")

	addEntry(t, service.Host())

	// Fetch data
	ms := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	events, err := mbtest.ReportingFetchV2Error(ms)
	if err != nil {
		t.Fatal("fetch", err)
	}

	t.Logf("%s/%s event: %+v", ms.Module().Name(), ms.Name(), events)

	// Make sure at least 1 db keyspace exists
	assert.True(t, len(events) > 0)

	keyspace := events[0].MetricSetFields

	assert.True(t, keyspace["avg_ttl"].(int64) >= 0)
	assert.True(t, keyspace["expires"].(int64) >= 0)
	assert.True(t, keyspace["keys"].(int64) >= 0)
	assert.True(t, strings.Contains(keyspace["id"].(string), "db"))
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "redis")

	addEntry(t, service.Host())

	ms := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	err := mbtest.WriteEventsReporterV2Error(ms, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}

// addEntry adds an entry to redis
func addEntry(t *testing.T, host string) {
	// Insert at least one event to make sure db exists
	c, err := rd.Dial("tcp", host)
	if err != nil {
		t.Fatal("connect", err)
	}
	defer c.Close()
	_, err = c.Do("SET", "foo", "bar")
	if err != nil {
		t.Fatal("SET", err)
	}
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "redis",
		"metricsets": []string{"keyspace"},
		"hosts":      []string{host},
	}
}
