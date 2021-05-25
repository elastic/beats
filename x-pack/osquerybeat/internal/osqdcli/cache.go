// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqdcli

type Cache interface {
	Add(key, value interface{}) (evicted bool)
	Get(key interface{}) (value interface{}, ok bool)
	Resize(size int) (evicted int)
}

func WithCache(cache Cache, minSize int) Option {
	return func(c *Client) {
		nsc := &nullSafeCache{cache: cache}
		if minSize > 0 {
			nsc.minSize = minSize
		}
		c.cache = nsc
	}
}

type nullSafeCache struct {
	cache   Cache
	minSize int
}

func (c *nullSafeCache) Add(key, value interface{}) (evicted bool) {
	if c.cache == nil {
		return
	}
	return c.cache.Add(key, value)
}

func (c *nullSafeCache) Get(key interface{}) (value interface{}, ok bool) {
	if c.cache == nil {
		return
	}
	return c.cache.Get(key)
}

func (c *nullSafeCache) Resize(size int) (evicted int) {
	if c.cache == nil {
		return
	}
	return c.cache.Resize(c.minSize + size)
}
