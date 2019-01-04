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

package key

import (
	"testing"

	rd "github.com/garyburd/redigo/redis"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/redis"
)

var host = redis.GetRedisEnvHost() + ":" + redis.GetRedisEnvPort()

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "redis")

	addEntry(t)

	ms := mbtest.NewReportingMetricSetV2(t, getConfig())
	events, err := mbtest.ReportingFetchV2(ms)
	if err != nil {
		t.Fatal("fetch", err)
	}

	t.Logf("%s/%s events: %+v", ms.Module().Name(), ms.Name(), events)
	assert.NotEmpty(t, events)
}

func TestData(t *testing.T) {
	compose.EnsureUp(t, "redis")

	addEntry(t)

	ms := mbtest.NewReportingMetricSetV2(t, getConfig())
	err := mbtest.WriteEventsReporterV2(ms, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}

// addEntry adds an entry to redis
func addEntry(t *testing.T) {
	// Insert at least one event to make sure db exists
	c, err := rd.Dial("tcp", host)
	if err != nil {
		t.Fatal("connect", err)
	}
	defer c.Close()
	_, err = c.Do("SET", "foo", "bar", "EX", "360")
	if err != nil {
		t.Fatal("SET", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "redis",
		"metricsets": []string{"key"},
		"hosts":      []string{host},
		"key.patterns": []map[string]interface{}{
			{
				"pattern": "foo",
			},
		},
	}
}
