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

package info

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/redis"

	"github.com/stretchr/testify/assert"
)

var redisHost = redis.GetRedisEnvHost() + ":" + redis.GetRedisEnvPort()

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "redis")

	ms := mbtest.NewReportingMetricSetV2(t, getConfig())
	events, err := mbtest.ReportingFetchV2(ms)
	if err != nil {
		t.Fatal("fetch", err)
	}
	if len(events) == 0 {
		t.Fatal("no events")
	}
	event := events[0].MetricSetFields

	t.Logf("%s/%s event: %+v", ms.Module().Name(), ms.Name(), event)

	// Check fields
	assert.Equal(t, 9, len(event))
	server := event["server"].(common.MapStr)
	assert.Equal(t, "standalone", server["mode"])
}

func TestData(t *testing.T) {
	compose.EnsureUp(t, "redis")

	ms := mbtest.NewReportingMetricSetV2(t, getConfig())
	err := mbtest.WriteEventsReporterV2(ms, t, "")
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "redis",
		"metricsets": []string{"info"},
		"hosts":      []string{redisHost},
	}
}
