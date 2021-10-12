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

package query

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/tests/compose"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "prometheus")

	config := map[string]interface{}{
		"module":     "prometheus",
		"metricsets": []string{"query"},
		"hosts":      []string{service.Host()},
		"queries": []common.MapStr{
			common.MapStr{
				"name": "go_threads",
				"path": "/api/v1/query",
				"params": common.MapStr{
					"query": "go_threads",
				},
			},
		},
	}
	ms := mbtest.NewReportingMetricSetV2Error(t, config)
	var err error
	for retries := 0; retries < 3; retries++ {
		err = mbtest.WriteEventsReporterV2Error(ms, t, "")
		if err == nil {
			return
		}
		time.Sleep(10 * time.Second)
	}
	t.Fatal("write", err)
}

func TestQueryFetch(t *testing.T) {
	service := compose.EnsureUp(t, "prometheus")

	config := map[string]interface{}{
		"module":     "prometheus",
		"metricsets": []string{"query"},
		"hosts":      []string{service.Host()},
		"queries": []common.MapStr{
			common.MapStr{
				"name": "go_info",
				"path": "/api/v1/query",
				"params": common.MapStr{
					"query": "go_info",
				},
			},
		},
	}
	f := mbtest.NewReportingMetricSetV2Error(t, config)

	var events []mb.Event
	var errors []error
	for retries := 0; retries < 3; retries++ {
		events, errors = mbtest.ReportingFetchV2Error(f)
		if len(events) > 0 {
			break
		}
		time.Sleep(10 * time.Second)
	}
	if len(errors) > 0 {
		t.Fatalf("Expected 0 errors, had %d. %v\n", len(errors), errors)
	}
	assert.NotEmpty(t, events)
	event := events[0].MetricSetFields
	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
}
