package virtualmachine

import (
	"strings"
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
		"metricsets": []string{"virtualmachine"},
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
	assert.True(t, strings.Contains(event["name"].(string), "ha-host_VM"))

	cpu := event["cpu"].(common.MapStr)

	cpuUsed := cpu["used"].(common.MapStr)
	assert.EqualValues(t, 0, cpuUsed["mhz"])

	memory := event["memory"].(common.MapStr)

	memoryUsed := memory["used"].(common.MapStr)
	memoryUsedHost := memoryUsed["host"].(common.MapStr)
	memoryUsedGuest := memoryUsed["guest"].(common.MapStr)
	assert.EqualValues(t, 0, memoryUsedGuest["bytes"])
	assert.EqualValues(t, 0, memoryUsedHost["bytes"])

	memoryTotal := memory["total"].(common.MapStr)
	memoryTotalGuest := memoryTotal["guest"].(common.MapStr)
	assert.EqualValues(t, uint64(33554432), memoryTotalGuest["bytes"])

	memoryFree := memory["free"].(common.MapStr)
	memoryFreeGuest := memoryFree["guest"].(common.MapStr)
	assert.EqualValues(t, uint64(33554432), memoryFreeGuest["bytes"])
}
