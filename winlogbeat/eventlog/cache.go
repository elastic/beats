package eventlog

// This component of the eventlog package provides a cache for storing Handles
// to event message files.

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/winlogbeat/sys/eventlogging"
)

// Constants that control the cache behavior.
const (
	expirationTimeout time.Duration = 2 * time.Minute
	janitorInterval   time.Duration = 30 * time.Second
	initialSize       int           = 10
)

// Function type for loading event message files associated with the given
// event log and source name.
type messageFileLoaderFunc func(eventLogName, sourceName string) eventlogging.MessageFiles

// Function type for freeing Handles.
type freeHandleFunc func(handle uintptr) error

// handleCache provides a synchronized cache that holds MessageFiles.
type messageFilesCache struct {
	cache        *common.Cache
	loader       messageFileLoaderFunc
	freer        freeHandleFunc
	eventLogName string
}

// newHandleCache creates and returns a new handleCache that has been
// initialized (including starting a periodic janitor goroutine to purge
// expired Handles).
func newMessageFilesCache(eventLogName string, loader messageFileLoaderFunc,
	freer freeHandleFunc) *messageFilesCache {

	hc := &messageFilesCache{
		loader:       loader,
		freer:        freer,
		eventLogName: eventLogName,
	}
	hc.cache = common.NewCacheWithRemovalListener(expirationTimeout,
		initialSize, hc.evictionHandler)
	hc.cache.StartJanitor(janitorInterval)
	return hc
}

// get returns a cached MessageFiles for the given sourceName.
// If no item is cached, then one is loaded, stored, and returned.
// Callers should check the MessageFiles.Err value to see if an error occurred
// while loading the message files.
func (hc *messageFilesCache) get(sourceName string) eventlogging.MessageFiles {
	v := hc.cache.Get(sourceName)
	if v == nil {
		// Handle to event message file for sourceName is not cached. Attempt
		// to load the Handles into the cache.
		v = hc.loader(hc.eventLogName, sourceName)

		// Store the newly loaded value. Since this code does not lock we must
		// check if a value was already loaded.
		existing := hc.cache.PutIfAbsent(sourceName, v)
		if existing != nil {
			// A value was already loaded, so free the handles we created.
			existingMessageFiles, _ := existing.(eventlogging.MessageFiles)
			hc.freeHandles(existingMessageFiles)
			return existingMessageFiles
		}
	}

	messageFiles, _ := v.(eventlogging.MessageFiles)
	return messageFiles
}

// evictionHandler is the callback handler that receives notifications when
// a key-value pair is evicted from the messageFilesCache.
func (hc *messageFilesCache) evictionHandler(k common.Key, v common.Value) {
	messageFiles, ok := v.(eventlogging.MessageFiles)
	if !ok {
		return
	}

	logp.Debug("eventlog", "Evicting messageFiles %+v for sourceName %v.",
		messageFiles, k)
	hc.freeHandles(messageFiles)
}

// freeHandles free the event message file Handles so that the modules can
// be unloaded. The Handles are no longer valid after being freed.
func (hc *messageFilesCache) freeHandles(mf eventlogging.MessageFiles) {
	for _, fh := range mf.Handles {
		err := hc.freer(fh.Handle)
		if err != nil {
			logp.Warn("FreeLibrary error for handle %v", fh.Handle)
		}
	}
}
