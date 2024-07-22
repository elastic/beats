// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v9

import (
	"bytes"
	"container/heap"
	"sync"
	"sync/atomic"
	"time"
)

type eventWithMissingTemplate struct {
	key       SessionKey
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
	*h = old[0 : n-1 : n-1]
	return x
}

type pendingTemplatesCache struct {
	mtx     sync.RWMutex
	wg      sync.WaitGroup
	isEmpty atomic.Bool
	hp      pendingEventsHeap
	started bool
	events  map[SessionKey][]*bytes.Buffer
}

func newPendingTemplatesCache() *pendingTemplatesCache {
	cache := &pendingTemplatesCache{
		events: make(map[SessionKey][]*bytes.Buffer),
		hp:     pendingEventsHeap{},
	}
	return cache
}

// GetAndRemove returns all events for a given session key and removes them from the cache
func (h *pendingTemplatesCache) GetAndRemove(key SessionKey) []*bytes.Buffer {
	if h == nil {
		return nil
	}

	if h.isEmpty.Load() {
		return nil
	}

	h.mtx.Lock()
	defer h.mtx.Unlock()
	events, ok := h.events[key]
	if !ok {
		return nil
	}
	delete(h.events, key)
	h.isEmpty.Store(len(h.events) == 0)
	return events
}

// Add adds an event to the pending templates cache
func (h *pendingTemplatesCache) Add(key SessionKey, events *bytes.Buffer) {
	if h == nil {
		return
	}

	h.mtx.Lock()
	defer h.mtx.Unlock()

	h.events[key] = append(h.events[key], events)
	h.hp.Push(eventWithMissingTemplate{key: key, entryTime: time.Now()})
	h.isEmpty.Store(false)
}

// start starts the pending templates cache cleaner
func (h *pendingTemplatesCache) start(done <-chan struct{}, cleanInterval time.Duration, removalThreshold time.Duration) {
	if h == nil {
		return
	}

	h.mtx.Lock()
	if h.started {
		h.mtx.Unlock()
		return
	}
	h.started = true
	h.mtx.Unlock()

	h.wg.Add(1)
	go func(n *pendingTemplatesCache) {
		defer n.wg.Done()
		ticker := time.NewTicker(cleanInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				n.mtx.Lock()
				if len(n.hp) == 0 {
					// lru is empty do not proceed further
					n.mtx.Unlock()
					continue
				} else if len(n.events) == 0 {
					// all pending events have been cleaned by GetAndRemove
					// thus reset lru since it is not empty (look above) and continue
					n.hp = pendingEventsHeap{}
					n.mtx.Unlock()
					continue
				}

				hp := &n.hp
				now := time.Now()
				for {
					v := heap.Pop(hp)
					c, ok := v.(eventWithMissingTemplate)
					if !ok {
						// weirdly enough we should never get here
						continue
					}
					if now.Sub(c.entryTime) < removalThreshold {
						// we have events that are not old enough
						// to be removed thus stop looping
						heap.Push(hp, c)
						break
					}
					// we can remove the pending events
					delete(n.events, c.key)
				}
				h.isEmpty.Store(len(h.events) == 0)
				n.mtx.Unlock()
			case <-done:
				return
			}
		}
	}(h)
}

// stop stops the pending templates cache cleaner
func (h *pendingTemplatesCache) stop() {
	if h == nil {
		return
	}

	h.wg.Wait()
}
