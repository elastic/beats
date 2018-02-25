package safemapstr

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/assert"
)

func TestPut(t *testing.T) {
	m := common.MapStr{
		"subMap": common.MapStr{
			"a": 1,
		},
	}

	// Add new value to the top-level.
	err := Put(m, "a", "ok")
	assert.NoError(t, err)
	assert.Equal(t, common.MapStr{"a": "ok", "subMap": common.MapStr{"a": 1}}, m)

	// Add new value to subMap.
	err = Put(m, "subMap.b", 2)
	assert.NoError(t, err)
	assert.Equal(t, common.MapStr{"a": "ok", "subMap": common.MapStr{"a": 1, "b": 2}}, m)

	// Overwrite a value in subMap.
	err = Put(m, "subMap.a", 2)
	assert.NoError(t, err)
	assert.Equal(t, common.MapStr{"a": "ok", "subMap": common.MapStr{"a": 2, "b": 2}}, m)

	// Add value to map that does not exist.
	m = common.MapStr{}
	err = Put(m, "subMap.newMap.a", 1)
	assert.NoError(t, err)
	assert.Equal(t, common.MapStr{"subMap": common.MapStr{"newMap": common.MapStr{"a": 1}}}, m)
}

func TestPutRenames(t *testing.T) {
	assert := assert.New(t)

	a := common.MapStr{}
	Put(a, "com.docker.swarm.task", "x")
	Put(a, "com.docker.swarm.task.id", 1)
	Put(a, "com.docker.swarm.task.name", "foobar")
	assert.Equal(common.MapStr{"com": common.MapStr{"docker": common.MapStr{"swarm": common.MapStr{
		"task": common.MapStr{
			"id":    1,
			"name":  "foobar",
			"value": "x",
		}}}}}, a)

	// order is not important:
	b := common.MapStr{}
	Put(b, "com.docker.swarm.task.id", 1)
	Put(b, "com.docker.swarm.task.name", "foobar")
	Put(b, "com.docker.swarm.task", "x")
	assert.Equal(common.MapStr{"com": common.MapStr{"docker": common.MapStr{"swarm": common.MapStr{
		"task": common.MapStr{
			"id":    1,
			"name":  "foobar",
			"value": "x",
		}}}}}, b)
}
