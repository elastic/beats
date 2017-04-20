// +build !integration

package node

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

	event, err := eventMapping(body)
	assert.NoError(t, err, "error mapping "+testFile)

	testCases := map[string]interface{}{
		"cpu.usage.core.ns":   4189523881380,
		"cpu.usage.nanocores": 18691146,

		"memory.available.bytes":  1768316928,
		"memory.usage.bytes":      2764943360,
		"memory.rss.bytes":        2150400,
		"memory.workingset.bytes": 2111090688,
		"memory.pagefaults":       131567,
		"memory.majorpagefaults":  103,

		"name": "gke-beats-default-pool-a5b33e2e-hdww",

		"fs.available.bytes": 98727014400,
		"fs.capacity.bytes":  101258067968,
		"fs.used.bytes":      2514276352,
		"fs.inodes.used":     138624,
		"fs.inodes.free":     6120096,
		"fs.inodes.count":    6258720,

		"network.rx.bytes":  1115133198,
		"network.rx.errors": 0,
		"network.tx.bytes":  812729002,
		"network.tx.errors": 0,

		"runtime.imagefs.available.bytes": 98727014400,
		"runtime.imagefs.capacity.bytes":  101258067968,
		"runtime.imagefs.used.bytes":      860204379,
	}

	for k, v := range testCases {
		testValue(t, event, k, v)
	}
}

func testValue(t *testing.T, event common.MapStr, field string, value interface{}) {
	data, err := event.GetValue(field)
	assert.NoError(t, err, "Could not read field "+field)
	assert.EqualValues(t, data, value, "Wrong value for field "+field)
}
