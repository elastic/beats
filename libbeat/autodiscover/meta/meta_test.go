package meta

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/assert"
)

func TestStoreNil(t *testing.T) {
	m := NewMap()
	assert.Equal(t, common.MapStrPointer{}, m.Store(0, nil))
}

func TestStore(t *testing.T) {
	m := NewMap()

	// Store meta
	res := m.Store(0, common.MapStr{"foo": "bar"})
	assert.Equal(t, res.Get(), common.MapStr{"foo": "bar"})

	// Update it
	res = m.Store(0, common.MapStr{"foo": "baz"})
	assert.Equal(t, res.Get(), common.MapStr{"foo": "baz"})

	m.Remove(0)
	assert.Equal(t, len(m.meta), 0)
}
