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

package dbstats

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "mongodb")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)

	for _, event := range events {
		t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
		metricsetFields := event.MetricSetFields

		// Check a few event Fields
		db := metricsetFields["db"].(string)
		assert.NotEqual(t, db, "")

		collections := metricsetFields["collections"].(int64)
		assert.True(t, collections > 0)

		objects := metricsetFields["objects"].(int64)
		assert.True(t, objects > 0)

		avgObjSize, err := metricsetFields.GetValue("avg_obj_size.bytes")
		assert.NoError(t, err)
		assert.True(t, avgObjSize.(int64) > 0)

		dataSize, err := metricsetFields.GetValue("data_size.bytes")
		assert.NoError(t, err)
		assert.True(t, dataSize.(int64) > 0)

		storageSize, err := metricsetFields.GetValue("storage_size.bytes")
		assert.NoError(t, err)
		assert.True(t, storageSize.(int64) > 0)

		numExtents := metricsetFields["num_extents"].(int64)
		assert.True(t, numExtents >= 0)

		indexes := metricsetFields["indexes"].(int64)
		assert.True(t, indexes >= 0)

		indexSize, err := metricsetFields.GetValue("index_size.bytes")
		assert.NoError(t, err)
		assert.True(t, indexSize.(int64) > 0)
	}
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "mongodb")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	if err := mbtest.WriteEventsReporterV2Error(f, t, ""); err != nil {
		t.Fatal("write", err)
	}
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "mongodb",
		"metricsets": []string{"dbstats"},
		"hosts":      []string{host},
	}
}
