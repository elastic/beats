// +build !integration

package container

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

const testFile = "../_meta/test/stats_summary.json"

func TestEventMapping(t *testing.T) {
	f, err := os.Open(testFile)
	assert.NoError(t, err, "cannot open test file "+testFile)

	body, err := ioutil.ReadAll(f)
	assert.NoError(t, err, "cannot read test file "+testFile)

	// TODO pass some state containers here
	events, err := eventMapping(body, []common.MapStr{
		common.MapStr{
			mb.NamespaceKey: "kubernetes.container",
			mb.ModuleDataKey: common.MapStr{
				"node": common.MapStr{
					"name": "gke-beats-default-pool-a5b33e2e-hdww",
				},
				"namespace": "default",
				"pod": common.MapStr{
					"name": "nginx-deployment-2303442956-pcqfc",
				},
			},
			"name": "nginx",
			"cpu": common.MapStr{
				"limit": common.MapStr{
					"cores": 2.0,
				},
			},
			"id":    "docker://39f3267ad1b0c46025e664bfe0b70f3f18a9f172aad00463c8e87e0e93bbf628",
			"image": "jenkinsci/jenkins:2.46.1",
			"memory": common.MapStr{
				"limit": common.MapStr{
					"bytes": 14622720.0,
				},
			},
			"status": common.MapStr{
				"phase":    "running",
				"ready":    true,
				"restarts": 4,
			},
		},
	})
	assert.NoError(t, err, "error mapping "+testFile)

	assert.Len(t, events, 1, "got wrong number of events")

	testCases := map[string]interface{}{
		"cpu.usage.core.ns":   43959424,
		"cpu.usage.nanocores": 11263994,

		"logs.available.bytes": 98727014400,
		"logs.capacity.bytes":  101258067968,
		"logs.used.bytes":      28672,
		"logs.inodes.count":    6258720,
		"logs.inodes.free":     6120096,
		"logs.inodes.used":     138624,

		"id": "docker://39f3267ad1b0c46025e664bfe0b70f3f18a9f172aad00463c8e87e0e93bbf628",

		"memory.available.bytes":  0,
		"memory.usage.bytes":      1462272,
		"memory.rss.bytes":        1409024,
		"memory.workingset.bytes": 1454080,
		"memory.pagefaults":       841,
		"memory.majorpagefaults":  0,

		"cpu.limit.cores":    2,
		"memory.limit.bytes": 14622720,

		// calculated pct fields:
		"cpu.usage.limit.pct":    0.005631997,
		"memory.usage.limit.pct": 0.1,

		"name": "nginx",

		"rootfs.available.bytes": 98727014400,
		"rootfs.capacity.bytes":  101258067968,
		"rootfs.used.bytes":      61440,
		"rootfs.inodes.used":     21,

		"status.phase":    "running",
		"status.ready":    true,
		"status.restarts": 4,
	}

	for k, v := range testCases {
		testValue(t, events[0], k, v)
	}
}

func testValue(t *testing.T, event common.MapStr, field string, value interface{}) {
	data, err := event.GetValue(field)
	assert.NoError(t, err, "Could not read field "+field)
	assert.EqualValues(t, data, value, "Wrong value for field "+field)
}
