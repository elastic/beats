package eventlog

// This component of the eventlog package provides a cache for storing Handles
// to event message files.

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Constants that control the cache behavior.
const (
	expirationTimeout time.Duration = 2 * time.Minute
	janitorInterval   time.Duration = 30 * time.Second
	initialSize       int           = 10
)

// Function type for loading event message file Handles associated with the
// given event log and source name.
type handleLoaderFunc func(eventLogName, sourceName string) ([]Handle, error)

// Function type for freeing Handles.
type freeLibraryFunc func(handle Handle) error

// This file provides a synchronized cache that holds the Handles to event
// message files which are either DLLs for EXEs.
type handleCache struct {
	cache        *common.Cache
	loader       handleLoaderFunc
	freer        freeLibraryFunc
	eventLogName string
}

// newHandleCache creates and returns a new handleCache that has been
// initialized (including starting a periodic janitor goroutine to purge
// expired Handles).
func newHandleCache(eventLogName string, loader handleLoaderFunc,
	freer freeLibraryFunc) *handleCache {

	hc := &handleCache{
		loader:       loader,
		freer:        freer,
		eventLogName: eventLogName,
	}
	hc.cache = common.NewCacheWithRemovalListener(expirationTimeout,
		initialSize, hc.evictionHandler)
	hc.cache.StartJanitor(janitorInterval)
	return hc
}

// get returns the cached event message file Handles for the given sourceName.
// If no Handles are cached, then the Handles are loaded, stored, and returned.
// If no event message files can be found for the sourceName then an empty
// array is returned.
func (hc *handleCache) get(sourceName string) []Handle {
	v := hc.cache.Get(sourceName)
	if v == nil {
		// Handle to event message file for sourceName is not cached. Attempt
		// to load the Handles into the cache.
		var err error
		v, err = hc.loader(hc.eventLogName, sourceName)
		if err != nil {
			// Cache the failure result as an empty slice.
			empty := []Handle{}
			hc.cache.PutIfAbsent(sourceName, empty)
			return empty
		}

		// Store the newly loaded value. Since this code does not lock we must
		// check if a value was already loaded.
		old := hc.cache.PutIfAbsent(sourceName, v)
		if old != nil {
			// A value was already loaded, so free the handles we created.
			oldHandles, _ := old.([]Handle)
			hc.freeHandles(oldHandles)
			return oldHandles
		}
	}

	handles, _ := v.([]Handle)
	return handles
}

// evictionHandler is the callback handler that receives notifications when
// a key-value pair is evicted from the handleCache.
func (hc *handleCache) evictionHandler(k common.Key, v common.Value) {
	handles, ok := v.([]Handle)
	if !ok {
		return
	}

	logp.Debug("eventlog", "Evicting handles %v for sourceName %v.", handles, k)
	hc.freeHandles(handles)
}

// freeHandles free the event message file Handles so that the modules can
// be unloaded. The Handles are no longer valid after being freed.
func (hc *handleCache) freeHandles(handles []Handle) {
	for _, handle := range handles {
		err := hc.freer(handle)
		if err != nil {
			logp.Warn("FreeLibrary error for handle %v", handle)
		}
	}
}
