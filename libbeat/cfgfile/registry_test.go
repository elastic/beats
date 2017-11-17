package cfgfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type runner struct{ id int }

func (runner) Start() {}
func (runner) Stop()  {}

func TestRegistryHas(t *testing.T) {
	registry := NewRegistry()

	registry.Add(1, runner{1})
	assert.True(t, registry.Has(1))
	assert.False(t, registry.Has(0))
}

func TestRegistryRemove(t *testing.T) {
	registry := NewRegistry()

	registry.Add(1, runner{1})
	assert.True(t, registry.Has(1))

	registry.Remove(1)
	assert.False(t, registry.Has(1))
}

func TestRegistryGet(t *testing.T) {
	registry := NewRegistry()

	registry.Add(1, runner{1})
	assert.Equal(t, registry.Get(1), runner{1})
}

func TestRegistryCopyList(t *testing.T) {
	registry := NewRegistry()

	registry.Add(1, runner{1})
	registry.Add(2, runner{2})

	list := registry.CopyList()
	registry.Remove(1)
	assert.Equal(t, len(list), 2)
}
