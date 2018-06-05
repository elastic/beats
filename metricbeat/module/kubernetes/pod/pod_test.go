// +build !integration

package pod

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/elastic/beats/metricbeat/mb"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

const testFile = "../_meta/test/stats_summary.json"

func TestEventMapping(t *testing.T) {
	f, err := os.Open(testFile)
	assert.NoError(t, err, "cannot open test file "+testFile)

	body, err := ioutil.ReadAll(f)
	assert.NoError(t, err, "cannot read test file "+testFile)

	events, err := eventMapping(body, []common.MapStr{
		common.MapStr{
			mb.NamespaceKey: "kubernetes.pod",
			mb.ModuleDataKey: common.MapStr{
				"node": common.MapStr{
					"name": "gke-beats-default-pool-a5b33e2e-hdww",
				},
				"namespace": "default",
			},
			"name":    "nginx-deployment-2303442956-pcqfc",
			"host_ip": "10.0.2.15",
			"status": common.MapStr{
				"ready":     "true",
				"phase":     "running",
				"scheduled": "true",
			},
		},
	})
	assert.NoError(t, err, "error mapping "+testFile)

	assert.Len(t, events, 1, "got wrong number of events")

	testCases := map[string]interface{}{
		"name": "nginx-deployment-2303442956-pcqfc",

		"network.rx.bytes":    107056,
		"network.rx.errors":   0,
		"network.tx.bytes":    72447,
		"network.tx.errors":   0,
		"cpu.usage.nanocores": 11263994,
		"memory.usage.bytes":  1462272,

		"status.ready":     "true",
		"status.phase":     "running",
		"status.scheduled": "true",
	}

	for k, v := range testCases {
		testValue(t, events[0], k, v)
	}
}

func testValue(t *testing.T, event common.MapStr, field string, expected interface{}) {
	data, err := event.GetValue(field)
	assert.NoError(t, err, "Could not read field "+field)
	assert.EqualValues(t, expected, data, "Wrong value for field "+field)
}
