package add_kubernetes_metadata

import (
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
)

type cache struct {
	sync.RWMutex
	timeout  time.Duration
	deleted  map[string]time.Time // key ->  when should this obj be deleted
	metadata map[string]common.MapStr
}

func newCache(cleanupTimeout time.Duration) *cache {
	c := &cache{
		timeout:  cleanupTimeout,
		deleted:  make(map[string]time.Time),
		metadata: make(map[string]common.MapStr),
	}
	go c.cleanup()
	return c
}

func (c *cache) get(key string) common.MapStr {
	c.Lock()
	defer c.Unlock()
	// add lifecycle if key was queried
	if t, ok := c.deleted[key]; ok {
		c.deleted[key] = t.Add(c.timeout)
	}
	return c.metadata[key]
}

func (c *cache) delete(key string) {
	c.Lock()
	defer c.Unlock()
	c.deleted[key] = time.Now().Add(c.timeout)
}

func (c *cache) set(key string, data common.MapStr) {
	c.Lock()
	defer c.Unlock()
	delete(c.deleted, key)
	c.metadata[key] = data
}

func (c *cache) cleanup() {
	ticker := time.Tick(timeout)
	space := []string{}
	for now := range ticker {
		shouldDelete := space[:0]
		c.RLock()
		for k, t := range c.deleted {
			if now.After(t) {
				shouldDelete = append(shouldDelete, k)
			}
		}
		c.RUnlock()
		c.Lock()
		for _, k := range shouldDelete {
			delete(c.deleted, k)
			delete(c.metadata, k)
		}
		c.Unlock()
	}
}
