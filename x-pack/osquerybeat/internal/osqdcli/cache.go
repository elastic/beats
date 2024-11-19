// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqdcli

type Cache[K comparable, V any] interface {
	Add(K, V) (evicted bool)
	Get(K) (value V, ok bool)
	Resize(size int) (evicted int)
}

func WithCache(cache Cache[string, map[string]string], minSize int) Option {
	return func(c *Client) {
		nsc := &nullSafeCache[string, map[string]string]{cache: cache}
		if minSize > 0 {
			nsc.minSize = minSize
		}
		c.cache = nsc
	}
}

type nullSafeCache[K comparable, V any] struct {
	cache   Cache[K, V]
	minSize int
}

func (c *nullSafeCache[K, V]) Add(key K, value V) (evicted bool) {
	if c.cache == nil {
		return false
	}
	return c.cache.Add(key, value)
}

func (c *nullSafeCache[K, V]) Get(key K) (value V, ok bool) {
	if c.cache == nil {
		return value, ok
	}
	return c.cache.Get(key)
}

func (c *nullSafeCache[K, V]) Resize(size int) (evicted int) {
	if c.cache == nil {
		return 0
	}
	return c.cache.Resize(c.minSize + size)
}
