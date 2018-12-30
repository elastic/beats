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
	"github.com/elastic/beats/metricbeat/module/mongodb"
)

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "mongodb")

	f := mbtest.NewEventsFetcher(t, getConfig())
	events, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	for _, event := range events {
		t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

		// Check a few event Fields
		db := event["db"].(string)
		assert.NotEqual(t, db, "")

		collections := event["collections"].(int64)
		assert.True(t, collections > 0)

		objects := event["objects"].(int64)
		assert.True(t, objects > 0)

		avgObjSize, err := event.GetValue("avg_obj_size.bytes")
		assert.NoError(t, err)
		assert.True(t, avgObjSize.(int64) > 0)

		dataSize, err := event.GetValue("data_size.bytes")
		assert.NoError(t, err)
		assert.True(t, dataSize.(int64) > 0)

		storageSize, err := event.GetValue("storage_size.bytes")
		assert.NoError(t, err)
		assert.True(t, storageSize.(int64) > 0)

		numExtents := event["num_extents"].(int64)
		assert.True(t, numExtents >= 0)

		indexes := event["indexes"].(int64)
		assert.True(t, indexes >= 0)

		indexSize, err := event.GetValue("index_size.bytes")
		assert.NoError(t, err)
		assert.True(t, indexSize.(int64) > 0)
	}
}

func TestData(t *testing.T) {
	compose.EnsureUp(t, "mongodb")

	f := mbtest.NewEventsFetcher(t, getConfig())
	err := mbtest.WriteEvents(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "mongodb",
		"metricsets": []string{"dbstats"},
		"hosts":      []string{mongodb.GetEnvHost() + ":" + mongodb.GetEnvPort()},
	}
}
