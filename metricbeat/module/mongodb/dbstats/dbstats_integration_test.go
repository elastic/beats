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

		dataSize := event["data_size"].(int64)
		assert.True(t, dataSize > 0)

		collections := event["collections"].(int64)
		assert.True(t, collections > 0)
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
