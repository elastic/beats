package queue

import (
	"container/heap"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestHeap(t *testing.T) {
	h := NewHeap()
	heap.Push(h, &item{
		priority: 13,
		value:    "foo",
	})

	heap.Push(h, &item{
		priority: 15,
		value:    "bar",
	})

	heap.Push(h, &item{
		priority: 12,
		value:    "xyz",
	})

	it := h.Pop().(*item)
	assert.Equal(t, it.priority, common.Float(15))
	assert.Equal(t, it.value, "bar")
	assert.Equal(t, h.Pop().(*item).priority, common.Float(13))
	assert.Equal(t, h.Pop().(*item).priority, common.Float(12))
}
