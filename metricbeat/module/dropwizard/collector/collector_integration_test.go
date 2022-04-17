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

package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/tests/compose"
	mbtest "github.com/menderesk/beats/v7/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "dropwizard")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)

	hasTag := false
	doesntHaveTag := false
	for _, event := range events {

		ok, _ := event.MetricSetFields.HasKey("my_histogram")
		if ok {
			_, err := event.MetricSetFields.GetValue("tags")
			if err == nil {
				t.Fatal("write", "my_counter not supposed to have tags")
			}
			doesntHaveTag = true
		}

		ok, _ = event.MetricSetFields.HasKey("my_counter")
		if ok {
			tagsRaw, err := event.MetricSetFields.GetValue("tags")
			if err != nil {
				t.Fatal("write", err)
			} else {
				tags, ok := tagsRaw.(common.MapStr)
				if !ok {
					t.Fatal("write", "unable to cast tags to common.MapStr")
				} else {
					assert.Equal(t, len(tags), 1)
					hasTag = true
				}
			}
		}
	}
	assert.Equal(t, hasTag, true)
	assert.Equal(t, doesntHaveTag, true)

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), events)
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":       "dropwizard",
		"metricsets":   []string{"collector"},
		"hosts":        []string{host},
		"namespace":    "testnamespace",
		"metrics_path": "/test/metrics",
		"enabled":      true,
	}
}
