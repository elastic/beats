// +build !integration

package container

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

const testFile = "../_meta/test/stats_summary.json"

func TestEventMapping(t *testing.T) {

	f, err := os.Open(testFile)
	assert.NoError(t, err, "cannot open test file "+testFile)

	body, err := ioutil.ReadAll(f)
	assert.NoError(t, err, "cannot read test file "+testFile)

	events, err := eventMapping(body)
	assert.NoError(t, err, "error mapping "+testFile)

	assert.Len(t, events, 1, "got wrong number of events")

	testCases := map[string]interface{}{
		"cpu.usage.core.ns":   43959424,
		"cpu.usage.nanocores": 0,

		"logs.available.bytes": 98727014400,
		"logs.capacity.bytes":  101258067968,
		"logs.used.bytes":      28672,
		"logs.inodes.count":    6258720,
		"logs.inodes.free":     6120096,
		"logs.inodes.used":     138624,

		"memory.available.bytes":  0,
		"memory.usage.bytes":      1462272,
		"memory.rss.bytes":        1409024,
		"memory.workingset.bytes": 1454080,
		"memory.pagefaults":       841,
		"memory.majorpagefaults":  0,

		"name": "nginx",

		"rootfs.available.bytes": 98727014400,
		"rootfs.capacity.bytes":  101258067968,
		"rootfs.used.bytes":      61440,
		"rootfs.inodes.used":     21,
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
