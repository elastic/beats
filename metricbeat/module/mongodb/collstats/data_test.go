// +build !integration

package collstats

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/assert"
)

func TestEventMapping(t *testing.T) {

	content, err := ioutil.ReadFile("./_meta/test/input.json")
	assert.NoError(t, err)

	data := common.MapStr{}
	json.Unmarshal(content, &data)

	event, _ := eventMapping("unit.test", data)

	assert.Equal(t, event["total"].(common.MapStr)["count"], float64(1))
}
