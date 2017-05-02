package util

import (
	"testing"

	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestNewData(t *testing.T) {

	data := NewData()

	assert.False(t, data.HasEvent())
	assert.False(t, data.HasState())

	data.SetState(file.State{Source: "-"})

	assert.False(t, data.HasEvent())
	assert.True(t, data.HasState())

	data.Event = common.MapStr{}

	assert.True(t, data.HasEvent())
	assert.True(t, data.HasState())
}

func TestGetEvent(t *testing.T) {

	data := NewData()
	data.Meta.Module = "testmodule"
	data.Meta.Fileset = "testfileset"
	data.Event = common.MapStr{"hello": "world"}

	out := common.MapStr{"fileset": common.MapStr{"module": "testmodule", "name": "testfileset"}, "hello": "world"}

	assert.Equal(t, out, data.GetEvent())
}
