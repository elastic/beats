package server

import "sync"

type cache struct {
	data    map[string]interface{}
	keylist []string
	idx     int
	maxSize int
	mtx     sync.RWMutex
}

func newCache(maxSize int) *cache {
	return &cache{
		data:    map[string]interface{}{},
		keylist: []string{},
		maxSize: maxSize,
	}
}

func (c *cache) Get(k string) (interface{}, bool) {
	c.mtx.RLock()
	v, ok := c.data[k]
	c.mtx.RUnlock()
	return v, ok
}

func (c *cache) Insert(k string, v interface{}) {

	// Short path if its already in the cache
	_, ok := c.Get(k)
	if ok {
		return
	}

	// Slow path, grab the write lock and insert
	c.mtx.Lock()
	_, ok = c.data[k]
	if !ok {
		c.data[k] = v
		if len(c.keylist) < c.maxSize {
			// Haven't reached max size yet, keep adding keys.
			c.keylist = append(c.keylist, k)
		} else {
			// Start recycling spots in the key list and
			// dropping cache entries for them.
			delete(c.data, c.keylist[c.idx])
			c.keylist[c.idx] = k
			c.idx = (c.idx + 1) % c.maxSize
		}
	}
	c.mtx.Unlock()
}
