package flowfilerepostorage

import (
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/metricbeat/module/nifi"
)

func TestFetch(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig())
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

	// Check event fields are present and correct
	freeSpace := event["free_space"].(string)
	assert.NotEqual(t, freeSpace, "")

	freeSpaceBytes := event["free_space_bytes"].(int64)
	assert.True(t, freeSpaceBytes > 0)

	totalSpace := event["total_space"].(string)
	assert.NotEqual(t, totalSpace, "")

	totalSpaceBytes := event["total_space_bytes"].(int64)
	assert.True(t, totalSpaceBytes > 0)

	usedSpace := event["used_space"].(string)
	assert.NotEqual(t, usedSpace, "")

	usedSpaceBytes := event["used_space_bytes"].(int64)
	assert.True(t, usedSpaceBytes > 0)

	utilization := event["utilization"].(string)
	assert.NotEqual(t, utilization, "")
}

func TestData(t *testing.T) {
	f := mbtest.NewEventFetcher(t, getConfig())
	err := mbtest.WriteEvent(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}

func getConfig() map[string]interface{} {
	return map[string]interface{}{
		"module":     "nifi",
		"metricsets": []string{"flowfilerepostorage"},
		"hosts":      []string{nifi.GetEnvHost() + ":" + nifi.GetEnvPort()},
	}
}
