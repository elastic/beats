// +build !integration

package system

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
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
		"container": "kubelet",

		"memory.usage.bytes":      36683776,
		"memory.rss.bytes":        35512320,
		"memory.workingset.bytes": 36495360,
		"memory.pagefaults":       100835242,
		"memory.majorpagefaults":  0,

		"cpu.usage.core.ns":   2357800908948,
		"cpu.usage.nanocores": 11263994,
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
