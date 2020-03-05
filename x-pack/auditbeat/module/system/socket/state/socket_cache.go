// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package state

import (
	"sync"
	"time"
)

type SocketRemovalListener func(v *Socket)
type socketWrapper struct {
	expiration time.Time
	value      *Socket
}

func (s *socketWrapper) IsExpired(now time.Time) bool {
	return now.After(s.expiration)
}
func (s *socketWrapper) UpdateLastAccessTime(now time.Time, expiration time.Duration) {
	s.expiration = now.Add(expiration)
}

type SocketCache struct {
	sync.RWMutex
	timeout      time.Duration              // Length of time before cache elements expire.
	closeTimeout time.Duration              // Length of time before cache elements expire.
	active       map[uintptr]*socketWrapper // Data stored by the cache.
	closing      map[uintptr]*socketWrapper // Data stored by the cache.
	listener     SocketRemovalListener      // Callback listen to notify of evictions.
}

func NewSocketCache(d, c time.Duration, l SocketRemovalListener) *SocketCache {
	return &SocketCache{
		timeout:      d,
		closeTimeout: c,
		active:       make(map[uintptr]*socketWrapper, 8),
		closing:      make(map[uintptr]*socketWrapper, 8),
		listener:     l,
	}
}

func (c *SocketCache) PutIfAbsent(v *Socket) *Socket {
	if v.socket == 0 {
		// uninitialized socket, no-op
		return nil
	}

	c.Lock()
	defer c.Unlock()
	oldValue := c.get(v.socket)
	if oldValue != nil {
		return oldValue
	}

	c.put(v.socket, v)
	return v
}

func (c *SocketCache) Get(k uintptr) *Socket {
	if k == 0 {
		// uninitialized socket, no-op
		return nil
	}

	c.RLock()
	defer c.RUnlock()
	return c.get(k)
}

func (c *SocketCache) Close(k uintptr) *Socket {
	if k == 0 {
		// uninitialized socket, no-op
		return nil
	}

	c.Lock()
	defer c.Unlock()
	return c.close(k)
}

func (c *SocketCache) CleanUp() (int, int) {
	c.Lock()
	defer c.Unlock()

	return c.cleanUpActive(), c.cleanUpClosed()
}

func (c *SocketCache) Size() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.active)
}

func (c *SocketCache) ClosingSize() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.closing)
}

func (c *SocketCache) cleanUpActive() int {
	count := 0
	for k, v := range c.active {
		if v.IsExpired(time.Now()) {
			c.close(k)
			count++
		}
	}
	return count
}

func (c *SocketCache) cleanUpClosed() int {
	count := 0
	for k, v := range c.closing {
		if v.IsExpired(time.Now()) {
			delete(c.closing, k)
			count++
			if c.listener != nil {
				c.listener(v.value)
			}
		}
	}
	return count
}

func (c *SocketCache) get(k uintptr) *Socket {
	if elem := c.getActive(k); elem != nil {
		return elem
	}
	return c.getClosing(k)
}

func (c *SocketCache) getClosing(k uintptr) *Socket {
	if elem := c.closing[k]; elem != nil && !elem.IsExpired(time.Now()) {
		return elem.value
	}
	return nil
}

func (c *SocketCache) getActive(k uintptr) *Socket {
	now := time.Now()
	if elem := c.active[k]; elem != nil && !elem.IsExpired(now) {
		elem.UpdateLastAccessTime(now, c.timeout)
		return elem.value
	}
	return nil
}

func (c *SocketCache) put(k uintptr, v *Socket) {
	if v == nil {
		panic("Cache does not support storing nil values.")
	}
	c.active[k] = &socketWrapper{
		expiration: time.Now().Add(c.timeout),
		value:      v,
	}
}

func (c *SocketCache) close(k uintptr) *Socket {
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
