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

package stats

import (
	"os"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "prometheus")

	f := mbtest.NewEventFetcher(t, getConfig())
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	// Check number of fields.
	assert.Equal(t, 3, len(event))
	assert.True(t, event["processes"].(common.MapStr)["open_fds"].(int64) > 0)
}

func TestData(t *testing.T) {
	compose.EnsureUp(t, "prometheus")

	f := mbtest.NewEventFetcher(t, getConfig())

	err := mbtest.WriteEvent(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "prometheus",
		"metricsets": []string{"stats"},
		"hosts":      []string{getPrometheusEnvHost() + ":" + getPrometheusEnvPort()},
	}
}

func getPrometheusEnvHost() string {
	host := os.Getenv("PROMETHEUS_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

func getPrometheusEnvPort() string {
	port := os.Getenv("PROMETHEUS_PORT")

	if len(port) == 0 {
		port = "9090"
	}
	return port
}
