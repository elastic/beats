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
