package nop

import (
	"testing"

	"github.com/elastic/libbeat/common"

	"github.com/stretchr/testify/assert"
)

func TestNopFilter(t *testing.T) {
	nop := new(Nop)
	plugin, err := nop.New("test", map[string]interface{}{})
	assert.Nil(t, err)

	x := common.MapStr{"me": "hello"}

	res, err := plugin.Filter(x)
	assert.Nil(t, err)
	assert.Equal(t, x, res)
}
