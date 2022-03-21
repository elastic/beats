package filesystem

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/stretchr/testify/assert"
)

func TestMountList(t *testing.T) {
	hostfs := resolve.NewTestResolver("/")

	result, err := GetFilesystems(hostfs, nil)
	assert.NoError(t, err, "GetFilesystems")

	// for _, res := range result {
	// 	t.Logf("FS: %#v\n", res)
	// }

	t.Logf("Usage:")

	for _, res := range result {
		err := res.getUsage()
		assert.NoError(t, err, "getUsage")
		out := common.MapStr{}
		typeconv.Convert(&out, res)
		t.Logf("Usage: %s", out.StringToPrint())
	}
}
