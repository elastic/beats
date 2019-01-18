// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cache

import (
	"testing"

	"github.com/OneOfOne/xxhash"
	"github.com/stretchr/testify/assert"
)

type CacheTestItem struct {
	s string
}

func (item CacheTestItem) Hash() uint64 {
	h := xxhash.New64()
	h.WriteString(item.s)
	return h.Sum64()
}

func TestCache(t *testing.T) {
	c := New()

	assert.True(t, c.IsEmpty())

	oldItems := []Cacheable{
		CacheTestItem{"item1"},
		CacheTestItem{"item2"},
	}

	newItems := []Cacheable{
		CacheTestItem{"item1"},
		CacheTestItem{"item3"},
	}

	new, missing := c.DiffAndUpdateCache(oldItems)

	assert.Equal(t, 2, len(new))
	assert.Equal(t, 0, len(missing))
	assert.False(t, c.IsEmpty())

	new, missing = c.DiffAndUpdateCache(newItems)

	assert.Equal(t, 1, len(new))
	assert.Equal(t, 1, len(missing))

	new, missing = c.DiffAndUpdateCache([]Cacheable{})

	assert.Equal(t, 0, len(new))
	assert.Equal(t, 2, len(missing))
	assert.True(t, c.IsEmpty())
}
