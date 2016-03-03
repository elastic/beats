package common

import (
	"sync"
	"time"
)

// Key type used in the cache.
type Key interface{}

// Value type held in the cache. Cannot be nil.
type Value interface{}

// RemovalListener is the callback function type that can be registered with
// the cache to receive notification of the removal of expired elements.
type RemovalListener func(k Key, v Value)

// Clock is the function type used to get the current time.
type clock func() time.Time

// element represents an element stored in the cache.
type element struct {
	expiration time.Time
	timeout    time.Duration
	value      Value
}

// IsExpired returns true if the element is expired (current time is greater
// than the expiration time).
func (e *element) IsExpired(now time.Time) bool {
	return now.After(e.expiration)
}

// UpdateLastAccessTime updates the expiration time of the element. This
// should be called each time the element is accessed.
func (e *element) UpdateLastAccessTime(now time.Time, expiration time.Duration) {
	e.expiration = now.Add(expiration)
}

// Cache is a semi-persistent mapping of keys to values. Elements added to the
// cache are store until they are explicitly deleted or are expired due time-
// based eviction based on last access time.
//
// Expired elements are not visible through classes methods, but they do remain
// stored in the cache until CleanUp() is invoked. Therefore CleanUp() must be
// invoked periodically to prevent the cache from becoming a memory leak. If
// you want to start a goroutine to perform periodic clean-up then see
// StartJanitor().
//
// Cache does not support storing nil values. Any attempt to put nil into
// the cache will cause a panic.
type Cache struct {
	sync.RWMutex
	timeout     time.Duration    // Length of time before cache elements expire.
	elements    map[Key]*element // Data stored by the cache.
	clock       clock            // Function used to get the current time.
	listener    RemovalListener  // Callback listen to notify of evictions.
	janitorQuit chan struct{}    // Closing this channel stop the janitor.
}

// NewCache creates and returns a new Cache. d is the length of time after last
// access that cache elements expire. initialSize is the initial allocation size
// used for the Cache's underlying map.
func NewCache(d time.Duration, initialSize int) *Cache {
	return newCache(d, initialSize, nil, time.Now)
}

// NewCacheWithRemovalListener creates and returns a new Cache and register a
// RemovalListener callback function. d is the length of time after last access
// that cache elements expire. initialSize is the initial allocation size used
// for the Cache's underlying map. l is the callback function that will be
// invoked when cache elements are removed from the map on CleanUp.
func NewCacheWithRemovalListener(d time.Duration, initialSize int, l RemovalListener) *Cache {
	return newCache(d, initialSize, l, time.Now)
}

func newCache(d time.Duration, initialSize int, l RemovalListener, t clock) *Cache {
	return &Cache{
		timeout:  d,
		elements: make(map[Key]*element, initialSize),
		listener: l,
		clock:    t,
	}
}

// PutIfAbsent writes the given key and value to the cache only if the key is
// absent from the cache. Nil is returned if the key-value pair were written,
// otherwise the old value is returned.
func (c *Cache) PutIfAbsent(k Key, v Value) Value {
	return c.PutIfAbsentWithTimeout(k, v, 0)
}

// PutIfAbsentWithTimeout writes the given key and value to the cache only if
// the key is absent from the cache. Nil is returned if the key-value pair were
// written, otherwise the old value is returned.
// The cache expiration time will be overwritten by timeout of the key being
// inserted.
func (c *Cache) PutIfAbsentWithTimeout(k Key, v Value, timeout time.Duration) Value {
	c.Lock()
	defer c.Unlock()
	oldValue, exists := c.get(k)
	if exists {
		return oldValue
	}

	c.put(k, v, timeout)
	return nil
}

// Put writes the given key and value to the map replacing any existing value
// if it exists. The previous value associated with the key returned or nil
// if the key was not present.
func (c *Cache) Put(k Key, v Value) Value {
	return c.PutWithTimeout(k, v, 0)
}

// PutWithTimeout writes the given key and value to the map replacing any
// existing value if it exists. The previous value associated with the key
// returned or nil if the key was not present.
// The cache expiration time will be overwritten by timeout of the key being
// inserted.
func (c *Cache) PutWithTimeout(k Key, v Value, timeout time.Duration) Value {
	c.Lock()
	defer c.Unlock()
	oldValue, _ := c.get(k)
	c.put(k, v, timeout)
	return oldValue
}

// Replace overwrites the value for a key only if the key exists. The old
// value is returned if the value is updated, otherwise nil is returned.
func (c *Cache) Replace(k Key, v Value) Value {
	return c.ReplaceWithTimeout(k, v, 0)
}

// ReplaceWithTimeout overwrites the value for a key only if the key exists. The
// old value is returned if the value is updated, otherwise nil is returned.
// The cache expiration time will be overwritten by timeout of the key being
// inserted.
func (c *Cache) ReplaceWithTimeout(k Key, v Value, timeout time.Duration) Value {
	c.Lock()
	defer c.Unlock()
	oldValue, exists := c.get(k)
	if !exists {
		return nil
	}

	c.put(k, v, timeout)
	return oldValue
}

// Get the current value associated with a key or nil if the key is not
// present. The last access time of the element is updated.
func (c *Cache) Get(k Key) Value {
	c.RLock()
	defer c.RUnlock()
	v, _ := c.get(k)
	return v
}

// Delete a key from the map and return the value or nil if the key does
// not exist. The RemovalListener is not notified for explicit deletions.
func (c *Cache) Delete(k Key) Value {
	c.Lock()
	defer c.Unlock()
	v, _ := c.get(k)
	delete(c.elements, k)
	return v
}

// CleanUp performs maintenance on the cache by removing expired elements from
// the cache. If a RemoveListener is registered it will be invoked for each
// element that is removed during this clean up operation. The RemovalListener
// is invoked on the caller's goroutine.
func (c *Cache) CleanUp() int {
	c.Lock()
	defer c.Unlock()
	count := 0
	for k, v := range c.elements {
		if v.IsExpired(c.clock()) {
			delete(c.elements, k)
			count++
			if c.listener != nil {
				c.listener(k, v.value)
			}
		}
	}
	return count
}

// Entries returns a shallow copy of the non-expired elements in the cache.
func (c *Cache) Entries() map[Key]Value {
	c.RLock()
	defer c.RUnlock()
	copy := make(map[Key]Value, len(c.elements))
	for k, v := range c.elements {
		if !v.IsExpired(c.clock()) {
			copy[k] = v.value
		}
	}
	return copy
}

// Size returns the number of elements in the cache. The number includes both
// active elements and expired elements that have not been cleaned up.
func (c *Cache) Size() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.elements)
}

// StartJanitor starts a goroutine that will periodically invoke the cache's
// CleanUp() method.
func (c *Cache) StartJanitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	c.janitorQuit = make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				c.CleanUp()
			case <-c.janitorQuit:
				ticker.Stop()
				return
			}
		}
	}()
}

// StopJanitor stops the goroutine created by StartJanitor.
func (c *Cache) StopJanitor() {
	close(c.janitorQuit)
}

// get returns the non-expired values from the cache.
func (c *Cache) get(k Key) (Value, bool) {
	elem, exists := c.elements[k]
	now := c.clock()
	if exists && !elem.IsExpired(now) {
		elem.UpdateLastAccessTime(now, elem.timeout)
		return elem.value, true
	}
	return nil, false
}

// put writes a key-value to the cache replacing any existing mapping.
func (c *Cache) put(k Key, v Value, timeout time.Duration) {
	if v == nil {
		panic("Cache does not support storing nil values.")
	}

	if timeout <= 0 {
		timeout = c.timeout
	}
	c.elements[k] = &element{
		expiration: c.clock().Add(timeout),
		timeout:    timeout,
		value:      v,
	}
}
