# BBolt as registry backend
 - Add a new backend for Filebeat's registry. The current backend
   interface is defined @libbeat/statestore/backend/backend.go and the
   current implementation is @@libbeat/statestore/backend/memlog
 - The registry can be chosen by setting `registry.type` in the
   configuration file.
 - The default is the new, bbolt, implementation.

# Design
The new design should be built like a 2 layer "caching system":

## In-memory hot storage with garbage collection.
It's a cache, where every record has a TTL:
 - We need to introduce a new config parameter registry.cache.ttl. 
 - There is a background garbage collector goroutine that removes
   expired entries only from the in-memory view.
 - The GC interval equals the entry TTL setting.
 - TTL is counted from the last access/change timestamp of the entry.
 - If an entry is frequently accessed – it does not get removed.
 - The value is expected to be set in minutes/hours, e.g. 1h.

## On-disk cold storage with garbage collection.
It’s a scalable on-disk storage (e.g. key-value) where we can
periodically export in-memory state changes (like before) and from
where we can later query a state by its key. This storage should also
have its own TTL for each entry:

 - We need to introduce a new config parameter registry.disk.ttl.
 - A background garbage collector goroutine should remove expired entries.
 - The GC interval equals the entry TTL setting.
 - TTL is counted from the last access/change timestamp of the entry.
 - This TTL should be the duration of inactivity for a file entry after which it’s considered stale and irrelevant.
 - The value is expected to be set in months/years.
 - It uses bbolt.

# Implementation steps
The implementation will be done, and tested, in two phases, first we
add the bbolt as a new backend for the store, ensure it works and
write some tests. Once we're happy with the implementation, we will
add the "in-memory hot storage".
