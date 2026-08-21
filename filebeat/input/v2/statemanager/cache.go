// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package statemanager

import (
	"context"
	"sync"
)

// Cache is a process-level singleton cache of values of type T keyed by string.
// Callers sharing the same key receive the same value instance and share a
// single background goroutine; resources are torn down only after the last
// caller has released its reference.
type Cache[T any] struct {
	mu      sync.Mutex
	entries map[string]*cacheEntry[T]
	closeFn func(T)
}

type cacheState uint8

const (
	cacheInitializing cacheState = iota
	cacheActive
	cacheDraining
)

type cacheEntry[T any] struct {
	state     cacheState
	ready     chan struct{}
	readyOnce sync.Once
	// drained is closed after cancel()+wg.Wait()+closeFn() complete. It is
	// only initialised when the entry transitions to cacheActive so that
	// concurrent Acquires that observe cacheDraining know when the entry has
	// been removed from the map and a fresh one can be opened.
	drained chan struct{}
	initErr error
	value   T
	users   int
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	key     string
}

// NewCache returns an empty Cache. closeFn is called once per key after all
// users have released and the background goroutine (if any) has returned.
// Pass nil if no cleanup is needed.
func NewCache[T any](closeFn func(T)) *Cache[T] {
	return &Cache[T]{
		entries: make(map[string]*cacheEntry[T]),
		closeFn: closeFn,
	}
}

// Acquire returns the shared value for key, opening it on first access.
//
// On the first call for a key, openFn creates the value. If runFn is non-nil,
// it is started in a background goroutine that must block until the supplied
// context is cancelled. Subsequent callers with the same key reuse the value.
// onWait, if non-nil, is called just before blocking for another goroutine's
// initialization to complete. onHit, if non-nil, is called when an existing
// active entry is reused. Both are suitable for logging or metrics.
//
// The returned release function must be called exactly once. When the last
// caller releases, the background goroutine is cancelled, Acquire waits for
// it to exit, and then calls closeFn (if set). A new Acquire for the same key
// during drain (between the last release and closeFn completing) waits for the
// drain to finish before opening a fresh entry, ensuring at most one writer is
// active for a given backend key at any moment.
func (c *Cache[T]) Acquire(
	key string,
	openFn func() (T, error),
	runFn func(ctx context.Context, value T),
	onWait func(),
	onHit func(),
) (value T, release func(), err error) {
	for {
		c.mu.Lock()
		e := c.entries[key]
		if e == nil {
			e = &cacheEntry[T]{
				key:   key,
				state: cacheInitializing,
				ready: make(chan struct{}),
			}
			c.entries[key] = e
			c.mu.Unlock()
			return c.initialize(e, openFn, runFn)
		}
		switch e.state {
		case cacheActive:
			e.users++
			v := e.value
			rel := c.newRelease(e)
			c.mu.Unlock()
			if onHit != nil {
				onHit()
			}
			return v, rel, nil
		case cacheInitializing:
			ready := e.ready
			c.mu.Unlock()
			if onWait != nil {
				onWait()
			}
			<-ready
			if e.initErr != nil {
				var zero T
				return zero, nil, e.initErr
			}
		case cacheDraining:
			// The entry is draining (cancel+wg.Wait+closeFn in progress).
			// Wait until the drain completes — at that point the entry is
			// removed from the map and a fresh one can be opened safely.
			drained := e.drained
			c.mu.Unlock()
			<-drained
		}
	}
}

func (c *Cache[T]) initialize(e *cacheEntry[T], openFn func() (T, error), runFn func(context.Context, T)) (T, func(), error) {
	value, err := openFn()

	c.mu.Lock()
	if err != nil {
		e.initErr = err
		delete(c.entries, e.key)
		c.mu.Unlock()
		e.readyOnce.Do(func() { close(e.ready) })
		var zero T
		return zero, nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	e.value = value
	e.cancel = cancel
	e.users = 1
	e.drained = make(chan struct{})
	e.state = cacheActive
	c.mu.Unlock()

	if runFn != nil {
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			runFn(ctx, value)
		}()
	}

	e.readyOnce.Do(func() { close(e.ready) })
	return value, c.newRelease(e), nil
}

// Len returns the number of active cache entries.
func (c *Cache[T]) Len() int {
	c.mu.Lock()
	n := len(c.entries)
	c.mu.Unlock()
	return n
}

func (c *Cache[T]) newRelease(e *cacheEntry[T]) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			c.mu.Lock()
			e.users--
			if e.users > 0 {
				c.mu.Unlock()
				return
			}
			e.state = cacheDraining
			cancel := e.cancel
			value := e.value
			c.mu.Unlock()

			if cancel != nil {
				cancel()
			}
			e.wg.Wait()

			if c.closeFn != nil {
				c.closeFn(value)
			}

			c.mu.Lock()
			delete(c.entries, e.key)
			c.mu.Unlock()

			close(e.drained)
		})
	}
}
