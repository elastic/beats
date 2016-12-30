// +build integration

package dbstats

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/mongodb"
	"github.com/stretchr/testify/assert"
)

func TestFetch(t *testing.T) {
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

		avgObjSize := event["avg_obj_size"].(int64)
		assert.True(t, avgObjSize > 0)

		dataSize := event["data_size"].(int64)
		assert.True(t, dataSize > 0)

		storageSize := event["storage_size"].(int64)
		assert.True(t, storageSize > 0)

		numExtents := event["num_extents"].(int64)
		assert.True(t, numExtents >= 0)

		indexes := event["indexes"].(int64)
		assert.True(t, indexes >= 0)

		indexSize := event["index_size"].(int64)
		assert.True(t, indexSize > 0)
	}
}

func TestData(t *testing.T) {
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
