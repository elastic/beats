// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v9

import (
	"bytes"
	"container/heap"
	"sync"
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
	return h[i].entryTime.Before(h[j].entryTime)
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
	h.mtx.Lock()
	defer h.mtx.Unlock()
	if len(h.events) == 0 {
		return nil
	}

	events, ok := h.events[key]
	if !ok {
		return nil
	}
	delete(h.events, key)
	return events
}

// Add adds an event to the pending templates cache
func (h *pendingTemplatesCache) Add(key SessionKey, events *bytes.Buffer) {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	h.events[key] = append(h.events[key], events)
	h.hp.Push(eventWithMissingTemplate{key: key, entryTime: time.Now()})
}

// start starts the pending templates cache cleaner
func (h *pendingTemplatesCache) start(done <-chan struct{}, cleanInterval time.Duration, removalThreshold time.Duration) {
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
		timer := time.NewTimer(cleanInterval)
		defer timer.Stop()

		for {
			select {
			case <-timer.C:
				h.cleanup(removalThreshold)
				timer.Reset(cleanInterval)
			case <-done:
				return
			}
		}
	}(h)
}

func (h *pendingTemplatesCache) cleanup(removalThreshold time.Duration) {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	if len(h.hp) == 0 {
		// lru is empty do not proceed further
		return
	} else if len(h.events) == 0 {
		// all pending events have been cleaned by GetAndRemove
		// thus reset lru since it is not empty (look above) and continue
		h.hp = pendingEventsHeap{}
		return
	}

	hp := &h.hp
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
		delete(h.events, c.key)

		if len(h.hp) == 0 {
			break
		}
	}
}

// stop stops the pending templates cache cleaner
func (h *pendingTemplatesCache) wait() {
	h.wg.Wait()
}
