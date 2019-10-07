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

package agent

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/tests/compose"
	"github.com/elastic/beats/metricbeat/module/consul"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	t.Skip("Skip flaky test on Consul Agent")
	logp.TestingSetup()

	service := compose.EnsureUp(t, "consul")

	f := mbtest.NewReportingMetricSetV2Error(t, consul.GetConfig([]string{"agent"}, service.Host()))
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)

	t.Logf("Found '%d' events", len(events))

	for _, event := range events {
		t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
		metricsetFields := event.MetricSetFields

		// Check runtime value
		runtime, ok := metricsetFields["runtime"].(common.MapStr)
		assert.True(t, ok)

		//Check heapObjects
		heapObjects, ok := runtime["heap_objects"].(float64)
		assert.True(t, ok)
		assert.True(t, heapObjects > float64(0))

	}
}
