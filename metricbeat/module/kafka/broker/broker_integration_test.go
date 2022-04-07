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

package broker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"

	// Register input module and metricset
	_ "github.com/elastic/beats/v8/metricbeat/module/jolokia"
	_ "github.com/elastic/beats/v8/metricbeat/module/jolokia/jmx"
)

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "kafka",
		compose.UpWithTimeout(600*time.Second),
		compose.UpWithAdvertisedHostEnvFileForPort(9092),
	)

	m := mbtest.NewFetcher(t, getConfig(service.HostForPort(8779)))
	m.WriteEvents(t, "")
}

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "kafka",
		compose.UpWithTimeout(600*time.Second),
		compose.UpWithAdvertisedHostEnvFileForPort(9092),
	)

	m := mbtest.NewFetcher(t, getConfig(service.HostForPort(8779)))
	events, errs := m.FetchEvents()
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	t.Logf("%s/%s event: %+v", m.Module().Name(), m.Name(), events[0])
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "kafka",
		"metricsets": []string{"broker"},
		"hosts":      []string{host},
	}
}
