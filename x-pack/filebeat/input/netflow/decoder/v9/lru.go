// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v9

import (
	"bytes"
	"container/heap"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
)

type eventWithMissingTemplate struct {
	setID     uint16
	entryTime time.Time
}

type pendingEventsHeap []eventWithMissingTemplate

func (h pendingEventsHeap) Len() int {
	return len(h)
}

func (h pendingEventsHeap) Less(i, j int) bool {
	return h[i].entryTime.Sub(h[j].entryTime) < 0
}

func (h pendingEventsHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *pendingEventsHeap) Push(x any) {
	v, ok := x.(eventWithMissingTemplate)
	if ok {
		*h = append(*h, v)
	}
}

func (h *pendingEventsHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type pendingTemplatesCache struct {
	mtx     sync.RWMutex
	isEmpty atomic.Bool
	hp      pendingEventsHeap
	events  map[uint16][]*bytes.Buffer
}

func newPendingTemplatesCache() *pendingTemplatesCache {
	cache := &pendingTemplatesCache{
		events: make(map[uint16][]*bytes.Buffer),
	}
	heap.Init(&cache.hp)
	return cache
}

func (h *pendingTemplatesCache) GetAndRemove(setID uint16) []*bytes.Buffer {
	if h == nil {
		return nil
	}

	if h.isEmpty.Load() {
		return nil
	}

	h.mtx.Lock()
	defer h.mtx.Unlock()
	events, ok := h.events[setID]
	if !ok {
		return nil
	}
	delete(h.events, setID)
	h.isEmpty.Store(len(h.events) == 0)
	return events
}

func (h *pendingTemplatesCache) Add(setID uint16, events *bytes.Buffer) {
	if h == nil {
		return
	}

	h.mtx.Lock()
	defer h.mtx.Unlock()

	h.events[setID] = append(h.events[setID], events)
	h.hp.Push(eventWithMissingTemplate{setID: setID, entryTime: time.Now()})
	h.isEmpty.Store(false)
}

// assumption will need to be revisited.
func (h *pendingTemplatesCache) start(done <-chan struct{}, cleanInterval time.Duration, removalThreshold time.Duration) {
	if h == nil {
		return
	}

	go func(n *pendingTemplatesCache) {
		ticker := time.NewTicker(cleanInterval)
		defer ticker.Stop()

		hp := &n.hp
		for {
			select {
			case <-ticker.C:
				n.mtx.Lock()
				now := time.Now()
				for {
					if len(n.events) == 0 {
						break
					}
					v := heap.Pop(hp)
					c, ok := v.(eventWithMissingTemplate)
					if !ok {
						// weirdly enough that we should never get here
						continue
					}
					if now.Sub(c.entryTime) < removalThreshold {
						// we have events that are not old enough
						// to be removed
						heap.Push(hp, c)
						break
					}
					_, ok = n.events[c.setID]
					if !ok {
						// pending events have already been cleaned
						continue
					}

					// we can remove the pending events
					delete(n.events, c.setID)
				}
				n.mtx.Unlock()
			case <-done:
				return
			}
		}
	}(h)
}
