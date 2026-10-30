// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"
)

type queuedItem interface {
	// rows is the number of events the item publishes.
	rows() int
	enqueued() time.Time
	// evictFirst reports whether overflow should drop the item before
	// any item for which it is false.
	evictFirst() bool
}

// rowQueue is a FIFO bounded by the total rows of its items, like the
// libbeat memory queue's events limit. On overflow it drops the oldest
// items instead of rejecting new ones, and it accepts an item larger than
// the limit whole once everything older has been dropped. Popped items no
// longer count against the limit.
type rowQueue[T queuedItem] struct {
	onEvict func(T)
	// observe, if set, is called with the queue lock held after every
	// change, with the queued items and rows.
	observe func(items, events int)
	notify  chan struct{}

	mu     sync.Mutex
	items  *list.List
	limit  int
	events int
	// first counts the queued evictFirst items, so evict scans only when
	// there is one, and firstRows counts their rows.
	first, firstRows int
}

// newRowQueue returns a queue holding up to maxEvents rows. onEvict is
// called, without the queue lock held, for each item dropped on overflow.
func newRowQueue[T queuedItem](maxEvents int, onEvict func(T)) *rowQueue[T] {
	return &rowQueue[T]{onEvict: onEvict, notify: make(chan struct{}, 1), items: list.New(), limit: maxEvents}
}

// push adds item, evicting older items until it fits. An evictFirst item
// that would need an item for which evictFirst is false evicted is dropped
// itself instead, without evicting anything. push never blocks.
func (q *rowQueue[T]) push(item T) {
	q.mu.Lock()
	first := item.evictFirst()
	if first && q.items.Len() > q.first && q.events-q.firstRows+item.rows() > q.limit {
		q.mu.Unlock()
		q.onEvict(item)
		return
	}
	var evicted []T
	for q.items.Len() > 0 && q.events+item.rows() > q.limit {
		evicted = append(evicted, q.evict())
	}
	q.items.PushBack(item)
	q.events += item.rows()
	if first {
		q.first++
		q.firstRows += item.rows()
	}
	q.changed()
	q.mu.Unlock()
	q.signal()
	q.evicted(evicted)
}

// evict removes the oldest evictFirst item, or the oldest item if there
// is none. The caller holds q.mu and ensures the queue is not empty.
func (q *rowQueue[T]) evict() T {
	victim := q.items.Front()
	for e := victim; q.first > 0 && e != nil; e = e.Next() {
		v, ok := e.Value.(T)
		if !ok {
			panic(fmt.Sprintf("rowQueue holds %T, want only items pushed as its element type", e.Value))
		}
		if v.evictFirst() {
			victim = e
			break
		}
	}
	return q.remove(victim)
}

func (q *rowQueue[T]) remove(e *list.Element) T {
	removed := q.items.Remove(e)
	item, ok := removed.(T)
	if !ok {
		panic(fmt.Sprintf("rowQueue holds %T, want only items pushed as its element type", removed))
	}
	q.events -= item.rows()
	if item.evictFirst() {
		q.first--
		q.firstRows -= item.rows()
	}
	return item
}

// changed reports the queue size to observe. The caller holds q.mu, so
// reports arrive in the order of the changes.
func (q *rowQueue[T]) changed() {
	if q.observe != nil {
		q.observe(q.items.Len(), q.events)
	}
}

func (q *rowQueue[T]) signal() {
	select {
	case q.notify <- struct{}{}:
	default:
	}
}

func (q *rowQueue[T]) evicted(items []T) {
	for _, item := range items {
		q.onEvict(item)
	}
}

// pop removes and returns the oldest item, waiting for one if the queue is
// empty. It returns false once ctx is done.
func (q *rowQueue[T]) pop(ctx context.Context) (T, bool) {
	for {
		if ctx.Err() != nil {
			var zero T
			return zero, false
		}
		q.mu.Lock()
		if q.items.Len() > 0 {
			item := q.remove(q.items.Front())
			more := q.items.Len() > 0
			q.changed()
			q.mu.Unlock()
			if more {
				// Pass the wake-up on in case another consumer consumed it.
				q.signal()
			}
			return item, true
		}
		q.mu.Unlock()
		select {
		case <-ctx.Done():
		case <-q.notify:
		}
	}
}

// requeue returns a popped item to the front of the queue, as its oldest
// item, evicting the oldest items until the queue fits or only one item
// remains.
func (q *rowQueue[T]) requeue(item T) {
	q.mu.Lock()
	q.items.PushFront(item)
	q.events += item.rows()
	if item.evictFirst() {
		q.first++
		q.firstRows += item.rows()
	}
	var evicted []T
	for q.items.Len() > 1 && q.events > q.limit {
		evicted = append(evicted, q.evict())
	}
	q.changed()
	q.mu.Unlock()
	q.evicted(evicted)
}

// setLimit changes the row limit, evicting the oldest items until the
// queue fits or only one item remains.
func (q *rowQueue[T]) setLimit(maxEvents int) {
	q.mu.Lock()
	q.limit = maxEvents
	var evicted []T
	for q.items.Len() > 1 && q.events > q.limit {
		evicted = append(evicted, q.evict())
	}
	q.changed()
	q.mu.Unlock()
	q.evicted(evicted)
}

// drain removes and returns all queued items, oldest first, without
// calling onEvict.
func (q *rowQueue[T]) drain() []T {
	q.mu.Lock()
	defer q.mu.Unlock()
	items := make([]T, 0, q.items.Len())
	for q.items.Len() > 0 {
		items = append(items, q.remove(q.items.Front()))
	}
	q.changed()
	return items
}

func (q *rowQueue[T]) stats() (items, events int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.items.Len(), q.events
}
