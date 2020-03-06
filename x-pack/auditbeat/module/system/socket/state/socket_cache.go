// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package state

import (
	"sync"
	"time"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/common"
)

type socketRemovalListener func(v *common.Socket)
type socketWrapper struct {
	expiration time.Time
	value      *common.Socket
}

func (s *socketWrapper) isExpired(now time.Time) bool {
	return now.After(s.expiration)
}
func (s *socketWrapper) updateLastAccessTime(now time.Time, expiration time.Duration) {
	s.expiration = now.Add(expiration)
}

type socketCache struct {
	sync.RWMutex
	timeout      time.Duration              // Length of time before cache elements expire.
	closeTimeout time.Duration              // Length of time before cache elements expire.
	active       map[uintptr]*socketWrapper // Data stored by the cache.
	closing      map[uintptr]*socketWrapper // Data stored by the cache.
	listener     socketRemovalListener      // Callback listen to notify of evictions.
}

func newSocketCache(d, c time.Duration, l socketRemovalListener) *socketCache {
	return &socketCache{
		timeout:      d,
		closeTimeout: c,
		active:       make(map[uintptr]*socketWrapper, 8),
		closing:      make(map[uintptr]*socketWrapper, 8),
		listener:     l,
	}
}

func (c *socketCache) PutIfAbsent(v *common.Socket) *common.Socket {
	if v.Key() == 0 {
		// uninitialized socket, no-op
		return nil
	}

	c.Lock()
	defer c.Unlock()
	oldValue := c.get(v.Key())
	if oldValue != nil {
		return oldValue
	}

	c.put(v.Key(), v)
	return v
}

func (c *socketCache) Get(k uintptr) *common.Socket {
	if k == 0 {
		// uninitialized socket, no-op
		return nil
	}

	c.RLock()
	defer c.RUnlock()
	return c.get(k)
}

func (c *socketCache) Close(k uintptr) *common.Socket {
	if k == 0 {
		// uninitialized socket, no-op
		return nil
	}

	c.Lock()
	defer c.Unlock()
	return c.close(k)
}

func (c *socketCache) CleanUp() (int, int) {
	c.Lock()
	defer c.Unlock()

	return c.cleanUpActive(), c.cleanUpClosed()
}

func (c *socketCache) Size() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.active)
}

func (c *socketCache) ClosingSize() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.closing)
}

func (c *socketCache) cleanUpActive() int {
	count := 0
	for k, v := range c.active {
		if v.isExpired(time.Now()) {
			c.close(k)
			count++
		}
	}
	return count
}

func (c *socketCache) cleanUpClosed() int {
	count := 0
	for k, v := range c.closing {
		if v.isExpired(time.Now()) {
			delete(c.closing, k)
			count++
			if c.listener != nil {
				c.listener(v.value)
			}
		}
	}
	return count
}

func (c *socketCache) get(k uintptr) *common.Socket {
	if elem := c.getActive(k); elem != nil {
		return elem
	}
	return c.getClosing(k)
}

func (c *socketCache) getClosing(k uintptr) *common.Socket {
	if elem := c.closing[k]; elem != nil && !elem.isExpired(time.Now()) {
		return elem.value
	}
	return nil
}

func (c *socketCache) getActive(k uintptr) *common.Socket {
	now := time.Now()
	if elem := c.active[k]; elem != nil && !elem.isExpired(now) {
		elem.updateLastAccessTime(now, c.timeout)
		return elem.value
	}
	return nil
}

func (c *socketCache) put(k uintptr, v *common.Socket) {
	if v == nil {
		panic("Cache does not support storing nil values.")
	}
	c.active[k] = &socketWrapper{
		expiration: time.Now().Add(c.timeout),
		value:      v,
	}
}

func (c *socketCache) close(k uintptr) *common.Socket {
	if elem := c.active[k]; elem != nil {
		delete(c.active, k)
		c.closing[k] = &socketWrapper{
			expiration: time.Now().Add(c.closeTimeout),
			value:      elem.value,
		}
		return elem.value
	}
	return nil
}
