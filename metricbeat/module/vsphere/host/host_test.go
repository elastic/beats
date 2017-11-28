package host

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/simulator"
)

func TestFetchEventContents(t *testing.T) {
	model := simulator.ESX()
	err := model.Create()
	if err := model.Create(); err != nil {
		t.Fatal(err)
	}

	ts := model.Service.NewServer()
	defer ts.Close()

	f := mbtest.NewEventsFetcher(t, getConfig(ts))

	events, err := f.Fetch()
	if err != nil {
		t.Fatal("fetch error", err)
	}

	event := events[0]

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event.StringToPrint())

	assert.EqualValues(t, "localhost.localdomain", event["name"])

	cpu := event["cpu"].(common.MapStr)

	cpuUsed := cpu["used"].(common.MapStr)
	assert.EqualValues(t, 67, cpuUsed["mhz"])

	cpuTotal := cpu["total"].(common.MapStr)
	assert.EqualValues(t, 4588, cpuTotal["mhz"])

	cpuFree := cpu["free"].(common.MapStr)
	assert.EqualValues(t, 4521, cpuFree["mhz"])

	memory := event["memory"].(common.MapStr)

	memoryUsed := memory["used"].(common.MapStr)
	assert.EqualValues(t, uint64(1472200704), memoryUsed["bytes"])

	memoryTotal := memory["total"].(common.MapStr)
	assert.EqualValues(t, uint64(4294430720), memoryTotal["bytes"])

	memoryFree := memory["free"].(common.MapStr)
	assert.EqualValues(t, uint64(2822230016), memoryFree["bytes"])
}

func TestData(t *testing.T) {
	model := simulator.ESX()
	if err := model.Create(); err != nil {
		t.Fatal(err)
	}

	ts := model.Service.NewServer()
	defer ts.Close()

	f := mbtest.NewEventsFetcher(t, getConfig(ts))

	if err := mbtest.WriteEvents(f, t); err != nil {
		t.Fatal("write", err)
	}
}

func getConfig(ts *simulator.Server) map[string]interface{} {
	urlSimulator := ts.URL.Scheme + "://" + ts.URL.Host + ts.URL.Path

	return map[string]interface{}{
		"module":     "vsphere",
		"metricsets": []string{"host"},
		"hosts":      []string{urlSimulator},
		"username":   "user",
		"password":   "pass",
		"insecure":   true,
	}
}
