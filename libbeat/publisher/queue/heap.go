package queue

import (
	"container/heap"
	"sync"

	"github.com/elastic/beats/libbeat/common"
)

// An item is something we manage in a priority queue.
type item struct {
	// The value of the item; arbitrary.
	value interface{}

	// The priority of the item in the queue.
	priority common.Float

	// The index of the item in the heap.
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int
}

type items []*item

func (it items) Len() int { return len(it) }

func (it items) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return it[i].priority > it[j].priority
}

func (it items) Swap(i, j int) {
	it[i], it[j] = it[j], it[i]
	it[i].index = i
	it[j].index = j
}

func (it *items) Push(x interface{}) {
	n := len(*it)
	item := x.(*item)
	item.index = n
	*it = append(*it, item)
}

func (it *items) Pop() interface{} {
	old := *it
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*it = old[0 : n-1]
	return item
}

// A priorityQueue implements heap.Interface and holds items.
type priorityQueue struct {
	items items
	sync.Mutex
}

func NewHeap() heap.Interface {
	pq := &priorityQueue{
		items: items{},
	}

	heap.Init(&pq.items)
	return pq
}

func (pq *priorityQueue) Len() int {
	pq.Lock()
	defer pq.Unlock()

	return pq.items.Len()
}

func (pq *priorityQueue) Less(i, j int) bool {
	pq.Lock()
	defer pq.Unlock()

	return pq.items.Less(i, j)
}

func (pq *priorityQueue) Swap(i, j int) {
	pq.Lock()
	defer pq.Unlock()

	pq.items.Swap(i, j)
}

func (pq *priorityQueue) Push(x interface{}) {
	pq.Lock()
	defer pq.Unlock()

	heap.Push(&pq.items, x)
}

func (pq *priorityQueue) Pop() interface{} {
	pq.Lock()
	defer pq.Unlock()

	return heap.Pop(&pq.items)
}
