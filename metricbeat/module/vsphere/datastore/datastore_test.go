package datastore

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/vic/pkg/vsphere/simulator"
)

func TestFetchEventContents(t *testing.T) {

	model := simulator.ESX()

	err := model.Create()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	ts := model.Service.NewServer()
	defer ts.Close()

	urlSimulator := ts.URL.Scheme + "://" + ts.URL.Host + ts.URL.Path

	config := map[string]interface{}{
		"module":     "vsphere",
		"metricsets": []string{"datastore"},
		"hosts":      []string{urlSimulator},
		"username":   "user",
		"password":   "pass",
		"insecure":   true,
	}

	f := mbtest.NewEventsFetcher(t, config)

	events, err := f.Fetch()

	event := events[0]
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	assert.EqualValues(t, "ha-datacenter", event["datacenter"])
	assert.EqualValues(t, "LocalDS_0", event["name"])
	assert.EqualValues(t, "local", event["fstype"])

	capacity := event["capacity"].(common.MapStr)

	// values are random
	capacityTotal := capacity["total"].(common.MapStr)
	assert.True(t, (capacityTotal["bytes"].(int64) > 1000000))

	capacityFree := capacity["free"].(common.MapStr)
	assert.True(t, (capacityFree["bytes"].(int64) > 1000000))

	capacityUsed := capacity["used"].(common.MapStr)
	assert.True(t, (capacityUsed["bytes"].(int64) > 1000000))
	assert.True(t, (capacityUsed["pct"].(int64) > 10))
}
