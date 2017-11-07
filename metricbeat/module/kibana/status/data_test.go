// +build !integration

package status

import (
	"io/ioutil"
	"testing"

	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/assert"
)

func TestEventMapping(t *testing.T) {
	content, err := ioutil.ReadFile("./_meta/test/input.json")
	assert.NoError(t, err)

	event := eventMapping(content)

	assert.Equal(t, event["metrics"].(common.MapStr)["concurrent_connections"], int64(12))
}
