# BBolt Registry Backend - Implementation Plan

## Executive Summary

This document provides a detailed implementation plan for adding a bbolt-based registry backend to Filebeat with a 2-layer caching system (in-memory hot cache + on-disk cold storage). Implementation follows a phased approach:
- **Phase 1**: bbolt backend with disk TTL/GC
- **Phase 2**: In-memory cache layer with TTL/GC

## 1. Requirements Analysis

### 1.1 Core Requirements

**Phase 1: BBolt Backend (Cold Storage)**
- Implement `backend.Registry` and `backend.Store` interfaces
- On-disk key-value storage using bbolt
- Disk-level TTL with garbage collection
- Configuration: `registry.type` (default: "bbolt"), `registry.disk.ttl`
- Thread-safe operations
- Background GC goroutine (interval = TTL)
- TTL based on last access/change timestamp

**Phase 2: In-Memory Cache (Hot Storage)**
- In-memory cache layer on top of bbolt
- Cache-level TTL with separate GC
- Configuration: `registry.cache.ttl`
- Background GC goroutine (interval = TTL)
- Transparent read-through/write-through caching
- Frequently accessed entries remain in cache

### 1.2 Interface Compliance

Must implement:
```go
// backend.Registry
type Registry interface {
    Access(name string) (Store, error)
    Close() error
}

// backend.Store
type Store interface {
    Close() error
    Has(key string) (bool, error)
    Get(key string, value interface{}) error
    Set(key string, value interface{}) error
    Remove(string) error
    Each(fn func(string, ValueDecoder) (bool, error)) error
    SetID(id string)
}
```

### 1.3 Design Patterns from Existing Backends

**memlog patterns:**
- In-memory state + persistent storage sync
- Background checkpoint operations
- Transaction ID tracking
- Graceful error recovery
- Thread-safety with RWMutex

**es patterns:**
- Simple Registry struct with mutex
- Context-aware operations
- Lazy initialization

## 2. Architecture Design

### 2.1 Phase 1: BBolt Backend Architecture

```
┌─────────────────────────────────────────────────┐
│              Registry                           │
│  - Manages multiple stores                     │
│  - Per-store bbolt DB files                    │
└─────────────────────────────────────────────────┘
                    │
                    ├─ Store "filebeat" (db file: filebeat.db)
                    ├─ Store "other" (db file: other.db)
                    └─ ...
                    
┌─────────────────────────────────────────────────┐
│              Store                              │
│  - bbolt.DB instance                           │
│  - RWMutex for thread-safety                   │
│  - Metadata tracking (access times, TTL)       │
└─────────────────────────────────────────────────┘
                    │
            ┌───────┴────────┐
            │   BBolt DB     │
            │  ┌──────────┐  │
            │  │  data    │  │ - Key-value pairs
            │  └──────────┘  │
            │  ┌──────────┐  │
            │  │ metadata │  │ - Access timestamps
            │  └──────────┘  │
            └────────────────┘

┌─────────────────────────────────────────────────┐
│          Disk GC Goroutine                      │
│  - Runs every registry.disk.ttl interval       │
│  - Scans all keys in metadata bucket           │
│  - Deletes entries with last_access_time >TTL  │
└─────────────────────────────────────────────────┘
```

### 2.2 Phase 2: Two-Layer Architecture

```
┌─────────────────────────────────────────────────┐
│              Registry                           │
│  - Manages multiple stores                     │
└─────────────────────────────────────────────────┘
                    │
┌─────────────────────────────────────────────────┐
│              Store (with cache)                 │
│  ┌───────────────────────────────────────────┐ │
│  │   In-Memory Hot Cache                     │ │
│  │  - map[string]cacheEntry                  │ │
│  │  - cacheEntry: {value, lastAccess}        │ │
│  │  - Mutex protected                        │ │
│  └───────────────────────────────────────────┘ │
│                    ↕ (read-through/write-through)
│  ┌───────────────────────────────────────────┐ │
│  │   BBolt Disk Storage                      │ │
│  │  - Persistent key-value store             │ │
│  │  - Metadata with timestamps               │ │
│  └───────────────────────────────────────────┘ │
└─────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────┐
│   Cache GC Goroutine                            │
│  - Runs every registry.cache.ttl interval      │
│  - Removes expired entries from cache only     │
│  - Does NOT delete from disk                   │
└─────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────┐
│   Disk GC Goroutine                             │
│  - Runs every registry.disk.ttl interval       │
│  - Deletes from both cache and disk            │
└─────────────────────────────────────────────────┘
```

### 2.3 Data Structures

**Phase 1: BBolt Buckets**
```go
// Bucket structure in bbolt:
// - "data" bucket: stores actual key-value pairs
//   - key: string (entry key)
//   - value: JSON-encoded entry value
//
// - "metadata" bucket: stores access timestamps
//   - key: string (entry key)
//   - value: JSON-encoded metadata
//     {
//       "last_access": <unix timestamp nano>,
//       "last_change": <unix timestamp nano>
//     }
```

**Phase 2: In-Memory Cache**
```go
type cacheEntry struct {
    value      map[string]interface{} // decoded value
    lastAccess time.Time
    mu         sync.RWMutex
}

type memCache struct {
    entries map[string]*cacheEntry
    mu      sync.RWMutex
}
```

### 2.4 TTL and GC Strategy

**Disk GC (Phase 1 & 2)**
- Interval: `registry.disk.ttl` (e.g., 30 days, 6 months)
- TTL calculation: `time.Now() - lastAccessTime > diskTTL`
- On expiry: delete from both metadata and data buckets
- Runs in background goroutine with ticker

**Cache GC (Phase 2)**
- Interval: `registry.cache.ttl` (e.g., 1h, 24h)
- TTL calculation: `time.Now() - lastAccessTime > cacheTTL`
- On expiry: remove from in-memory cache only (keep on disk)
- Access-based: frequently accessed entries stay in cache

**Access Time Updates**
- Update on: Get, Set
- NOT updated on: Has, Remove, Each
- Persisted to disk on every access (can be optimized with batching)

### 2.5 Thread Safety

**Registry Level**
- Mutex protects store map and active flag
- Similar to memlog pattern

**Store Level**
- RWMutex for read/write operations
- Read operations: Has, Get, Each
- Write operations: Set, Remove
- Separate lock for cache vs disk (Phase 2)

**GC Goroutines**
- Acquire write lock when deleting entries
- Batch operations to minimize lock time

## 3. Configuration Schema

### 3.1 Configuration Structure

```go
// filebeat/config/config.go
type Registry struct {
    // Existing fields
    Path          string        `config:"path"`
    Permissions   os.FileMode   `config:"file_permissions"`
    FlushTimeout  time.Duration `config:"flush"`
    CleanInterval time.Duration `config:"cleanup_interval"`
    MigrateFile   string        `config:"migrate_file"`
    
    // NEW: Backend type selection
    Type          string        `config:"type"` // "memlog", "bbolt", "es"
    
    // NEW: BBolt-specific settings
    BBolt         BBoltConfig   `config:"bbolt"`
}

type BBoltConfig struct {
    // Disk storage TTL
    DiskTTL       time.Duration `config:"disk_ttl"`
    
    // Cache TTL (Phase 2)
    CacheTTL      time.Duration `config:"cache_ttl"`
    
    // BBolt-specific options
    FileMode      os.FileMode   `config:"file_permissions"`
    Timeout       time.Duration `config:"timeout"`
    NoGrowSync    bool          `config:"no_grow_sync"`
    NoFreelistSync bool         `config:"no_freelist_sync"`
}

var DefaultConfig = Config{
    Registry: Registry{
        Type: "bbolt", // NEW DEFAULT
        BBolt: BBoltConfig{
            DiskTTL:        30 * 24 * time.Hour,  // 30 days
            CacheTTL:       1 * time.Hour,         // 1 hour (Phase 2)
            FileMode:       0o600,
            Timeout:        1 * time.Second,
            NoGrowSync:     false,
            NoFreelistSync: true, // Performance optimization
        },
        // ... existing defaults
    },
}
```

### 3.2 YAML Configuration Examples

```yaml
# Example 1: Use bbolt with defaults
filebeat.registry:
  type: bbolt

# Example 2: BBolt with custom TTLs
filebeat.registry:
  type: bbolt
  bbolt:
    disk_ttl: 60d    # 60 days
    cache_ttl: 2h    # 2 hours (Phase 2)

# Example 3: Legacy memlog backend
filebeat.registry:
  type: memlog
  path: registry
  
# Example 4: ES backend (existing)
filebeat.registry:
  type: es
```

## 4. File Structure

### 4.1 New Files to Create

```
libbeat/statestore/backend/bbolt/
├── registry.go           # Registry implementation
├── store.go             # Store implementation (Phase 1)
├── store_cache.go       # Cache layer (Phase 2)
├── gc.go                # Garbage collection logic
├── metadata.go          # Metadata handling
├── doc.go              # Package documentation
├── bbolt_test.go       # Compliance tests
├── store_test.go       # Unit tests
├── gc_test.go          # GC tests
└── testdata/           # Test fixtures
```

### 4.2 Files to Modify

```
filebeat/config/config.go
  - Add Type field to Registry struct
  - Add BBoltConfig struct
  - Update DefaultConfig

filebeat/beater/store.go
  - Modify openStateStore() to support registry type selection
  - Add bbolt registry initialization
  - Keep backward compatibility with memlog

libbeat/statestore/backend/backend.go
  - No changes needed (interface stays same)

go.mod
  - Already has go.etcd.io/bbolt v1.4.0
```

## 5. Implementation Details

### 5.1 Phase 1: BBolt Backend Implementation

#### 5.1.1 Registry Implementation

**File:** `libbeat/statestore/backend/bbolt/registry.go`

```go
package bbolt

import (
    "os"
    "path/filepath"
    "sync"
    "time"

    "github.com/elastic/beats/v7/libbeat/statestore/backend"
    "github.com/elastic/elastic-agent-libs/logp"
)

type Registry struct {
    log    *logp.Logger
    mu     sync.Mutex
    active bool
    
    settings Settings
    
    // Active stores
    stores map[string]*store
    
    // GC control
    gcDone chan struct{}
    gcWg   sync.WaitGroup
}

type Settings struct {
    Root           string
    FileMode       os.FileMode
    DiskTTL        time.Duration
    Timeout        time.Duration
    NoGrowSync     bool
    NoFreelistSync bool
}

func New(log *logp.Logger, settings Settings) (*Registry, error) {
    // Validate settings
    // Create root directory
    // Initialize registry
    // Start disk GC goroutine
}

func (r *Registry) Access(name string) (backend.Store, error) {
    // Check if store already open
    // Open/create bbolt DB file
    // Return store instance
}

func (r *Registry) Close() error {
    // Stop GC goroutine
    // Close all open stores
    // Wait for cleanup
}

func (r *Registry) runDiskGC() {
    // Background goroutine
    // Ticker based on diskTTL
    // Iterate stores and call GC
}
```

#### 5.1.2 Store Implementation

**File:** `libbeat/statestore/backend/bbolt/store.go`

```go
package bbolt

import (
    "encoding/json"
    "time"
    
    "go.etcd.io/bbolt"
    "github.com/elastic/beats/v7/libbeat/statestore/backend"
)

type store struct {
    db   *bbolt.DB
    log  *logp.Logger
    mu   sync.RWMutex
    
    name     string
    settings Settings
    
    closed bool
}

const (
    dataBucket     = "data"
    metadataBucket = "metadata"
)

type metadata struct {
    LastAccess time.Time `json:"last_access"`
    LastChange time.Time `json:"last_change"`
}

type entry struct {
    value map[string]interface{}
}

func openStore(log *logp.Logger, path string, settings Settings) (*store, error) {
    // Open bbolt database
    // Create buckets if needed
    // Return store instance
}

func (s *store) Close() error {
    // Close bbolt DB
    // Mark as closed
}

func (s *store) Has(key string) (bool, error) {
    // Read-only transaction
    // Check if key exists in data bucket
}

func (s *store) Get(key string, value interface{}) error {
    // Read-only transaction
    // Read from data bucket
    // Update access time in metadata bucket (requires write tx)
    // Decode into value
}

func (s *store) Set(key string, value interface{}) error {
    // Write transaction
    // Encode value to JSON
    // Store in data bucket
    // Update metadata (last_access, last_change)
}

func (s *store) Remove(key string) error {
    // Write transaction
    // Delete from data bucket
    // Delete from metadata bucket
}

func (s *store) Each(fn func(string, backend.ValueDecoder) (bool, error)) error {
    // Read-only transaction
    // Iterate data bucket
    // Call fn for each entry
}

func (s *store) SetID(id string) {
    // NOOP or store in metadata
}

func (e entry) Decode(to interface{}) error {
    // Use typeconv.Convert
}
```

#### 5.1.3 Garbage Collection

**File:** `libbeat/statestore/backend/bbolt/gc.go`

```go
package bbolt

import (
    "time"
    
    "go.etcd.io/bbolt"
    "github.com/elastic/elastic-agent-libs/logp"
)

func (s *store) runGC(interval time.Duration, done <-chan struct{}) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-done:
            return
        case <-ticker.C:
            if err := s.collectGarbage(); err != nil {
                s.log.Errorf("GC failed: %v", err)
            }
        }
    }
}

func (s *store) collectGarbage() error {
    now := time.Now()
    var toDelete []string
    
    // Phase 1: Identify expired entries
    err := s.db.View(func(tx *bbolt.Tx) error {
        metaBucket := tx.Bucket([]byte(metadataBucket))
        if metaBucket == nil {
            return nil
        }
        
        return metaBucket.ForEach(func(k, v []byte) error {
            var meta metadata
            if err := json.Unmarshal(v, &meta); err != nil {
                return err
            }
            
            if now.Sub(meta.LastAccess) > s.settings.DiskTTL {
                toDelete = append(toDelete, string(k))
            }
            return nil
        })
    })
    
    if err != nil {
        return err
    }
    
    // Phase 2: Delete expired entries
    if len(toDelete) > 0 {
        s.log.Infof("GC: deleting %d expired entries", len(toDelete))
        
        err = s.db.Update(func(tx *bbolt.Tx) error {
            dataBucket := tx.Bucket([]byte(dataBucket))
            metaBucket := tx.Bucket([]byte(metadataBucket))
            
            for _, key := range toDelete {
                if dataBucket != nil {
                    dataBucket.Delete([]byte(key))
                }
                if metaBucket != nil {
                    metaBucket.Delete([]byte(key))
                }
            }
            return nil
        })
    }
    
    return err
}
```

#### 5.1.4 Metadata Handling

**File:** `libbeat/statestore/backend/bbolt/metadata.go`

```go
package bbolt

import (
    "encoding/json"
    "time"
    
    "go.etcd.io/bbolt"
)

func (s *store) updateAccessTime(tx *bbolt.Tx, key string) error {
    bucket := tx.Bucket([]byte(metadataBucket))
    if bucket == nil {
        return nil
    }
    
    var meta metadata
    if v := bucket.Get([]byte(key)); v != nil {
        json.Unmarshal(v, &meta)
    }
    
    meta.LastAccess = time.Now()
    
    data, err := json.Marshal(meta)
    if err != nil {
        return err
    }
    
    return bucket.Put([]byte(key), data)
}

func (s *store) updateMetadata(tx *bbolt.Tx, key string, changeTime bool) error {
    bucket := tx.Bucket([]byte(metadataBucket))
    if bucket == nil {
        return nil
    }
    
    now := time.Now()
    meta := metadata{
        LastAccess: now,
    }
    
    if changeTime {
        meta.LastChange = now
    } else {
        // Preserve existing change time
        if v := bucket.Get([]byte(key)); v != nil {
            var existing metadata
            json.Unmarshal(v, &existing)
            meta.LastChange = existing.LastChange
        }
    }
    
    data, err := json.Marshal(meta)
    if err != nil {
        return err
    }
    
    return bucket.Put([]byte(key), data)
}
```

### 5.2 Phase 2: In-Memory Cache Layer

#### 5.2.1 Cache Store Implementation

**File:** `libbeat/statestore/backend/bbolt/store_cache.go`

```go
package bbolt

import (
    "sync"
    "time"
)

type cacheEntry struct {
    value      map[string]interface{}
    lastAccess time.Time
    mu         sync.RWMutex
}

type storeWithCache struct {
    *store // Embed Phase 1 store
    
    cache   map[string]*cacheEntry
    cacheMu sync.RWMutex
    
    cacheTTL time.Duration
    
    // GC control
    cacheGCDone chan struct{}
    cacheGCWg   sync.WaitGroup
}

func openStoreWithCache(log *logp.Logger, path string, settings Settings) (*storeWithCache, error) {
    // Open base store
    baseStore, err := openStore(log, path, settings)
    if err != nil {
        return nil, err
    }
    
    s := &storeWithCache{
        store:       baseStore,
        cache:       make(map[string]*cacheEntry),
        cacheTTL:    settings.CacheTTL,
        cacheGCDone: make(chan struct{}),
    }
    
    // Start cache GC goroutine
    s.cacheGCWg.Add(1)
    go func() {
        defer s.cacheGCWg.Done()
        s.runCacheGC(settings.CacheTTL, s.cacheGCDone)
    }()
    
    return s, nil
}

func (s *storeWithCache) Close() error {
    // Stop cache GC
    close(s.cacheGCDone)
    s.cacheGCWg.Wait()
    
    // Close underlying store
    return s.store.Close()
}

func (s *storeWithCache) Get(key string, value interface{}) error {
    // Try cache first
    s.cacheMu.RLock()
    cached, found := s.cache[key]
    s.cacheMu.RUnlock()
    
    if found {
        cached.mu.Lock()
        cached.lastAccess = time.Now()
        cached.mu.Unlock()
        
        return entry{value: cached.value}.Decode(value)
    }
    
    // Cache miss - read from disk
    var decoded map[string]interface{}
    if err := s.store.Get(key, &decoded); err != nil {
        return err
    }
    
    // Populate cache
    s.cacheMu.Lock()
    s.cache[key] = &cacheEntry{
        value:      decoded,
        lastAccess: time.Now(),
    }
    s.cacheMu.Unlock()
    
    return entry{value: decoded}.Decode(value)
}

func (s *storeWithCache) Set(key string, value interface{}) error {
    // Write to disk first
    if err := s.store.Set(key, value); err != nil {
        return err
    }
    
    // Update cache
    var decoded map[string]interface{}
    typeconv.Convert(&decoded, value)
    
    s.cacheMu.Lock()
    s.cache[key] = &cacheEntry{
        value:      decoded,
        lastAccess: time.Now(),
    }
    s.cacheMu.Unlock()
    
    return nil
}

func (s *storeWithCache) Remove(key string) error {
    // Remove from disk
    if err := s.store.Remove(key); err != nil {
        return err
    }
    
    // Remove from cache
    s.cacheMu.Lock()
    delete(s.cache, key)
    s.cacheMu.Unlock()
    
    return nil
}

func (s *storeWithCache) runCacheGC(interval time.Duration, done <-chan struct{}) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-done:
            return
        case <-ticker.C:
            s.collectCacheGarbage()
        }
    }
}

func (s *storeWithCache) collectCacheGarbage() {
    now := time.Now()
    var toDelete []string
    
    s.cacheMu.RLock()
    for key, entry := range s.cache {
        entry.mu.RLock()
        if now.Sub(entry.lastAccess) > s.cacheTTL {
            toDelete = append(toDelete, key)
        }
        entry.mu.RUnlock()
    }
    s.cacheMu.RUnlock()
    
    if len(toDelete) > 0 {
        s.cacheMu.Lock()
        for _, key := range toDelete {
            delete(s.cache, key)
        }
        s.cacheMu.Unlock()
        
        s.log.Debugf("Cache GC: removed %d entries", len(toDelete))
    }
}
```

### 5.3 Backend Selection Logic

**File:** `filebeat/beater/store.go` (modification)

```go
func openStateStore(ctx context.Context, info beat.Info, logger *logp.Logger, cfg config.Registry, beatPaths *paths.Path) (*filebeatStore, error) {
    var (
        reg backend.Registry
        err error

        esreg    *es.Registry
        notifier *es.Notifier
    )

    if features.IsElasticsearchStateStoreEnabled() {
        notifier = es.NewNotifier()
        esreg = es.New(ctx, logger, notifier)
    }

    // NEW: Backend type selection
    switch cfg.Type {
    case "bbolt", "": // Empty defaults to bbolt
        reg, err = bbolt.New(logger, bbolt.Settings{
            Root:           beatPaths.Resolve(paths.Data, cfg.Path),
            FileMode:       cfg.BBolt.FileMode,
            DiskTTL:        cfg.BBolt.DiskTTL,
            CacheTTL:       cfg.BBolt.CacheTTL,
            Timeout:        cfg.BBolt.Timeout,
            NoGrowSync:     cfg.BBolt.NoGrowSync,
            NoFreelistSync: cfg.BBolt.NoFreelistSync,
        })
    case "memlog":
        reg, err = memlog.New(logger, memlog.Settings{
            Root:     beatPaths.Resolve(paths.Data, cfg.Path),
            FileMode: cfg.Permissions,
        })
    default:
        return nil, fmt.Errorf("unknown registry backend type: %s", cfg.Type)
    }
    
    if err != nil {
        return nil, err
    }

    store := &filebeatStore{
        registry:      statestore.NewRegistry(reg),
        storeName:     info.Beat,
        cleanInterval: cfg.CleanInterval,
        notifier:      notifier,
    }

    if esreg != nil {
        store.esRegistry = statestore.NewRegistry(esreg)
    }

    return store, nil
}
```

## 6. Testing Strategy

### 6.1 Compliance Tests

**File:** `libbeat/statestore/backend/bbolt/bbolt_test.go`

```go
package bbolt

import (
    "testing"
    
    "github.com/elastic/beats/v7/libbeat/statestore/internal/storecompliance"
)

func TestBBoltCompliance(t *testing.T) {
    storecompliance.TestBackendCompliance(t, func(testPath string) (backend.Registry, error) {
        return New(logp.NewLogger("bbolt"), Settings{
            Root:       testPath,
            FileMode:   0600,
            DiskTTL:    24 * time.Hour,
            CacheTTL:   1 * time.Hour, // Phase 2
            Timeout:    1 * time.Second,
        })
    })
}

func TestBBoltComplianceWithCache(t *testing.T) {
    // Phase 2: Test with cache enabled
    storecompliance.TestBackendCompliance(t, func(testPath string) (backend.Registry, error) {
        return New(logp.NewLogger("bbolt"), Settings{
            Root:       testPath,
            FileMode:   0600,
            DiskTTL:    24 * time.Hour,
            CacheTTL:   1 * time.Hour,
            Timeout:    1 * time.Second,
        })
    })
}
```

### 6.2 Unit Tests

**File:** `libbeat/statestore/backend/bbolt/store_test.go`

Test cases:
- Store creation and initialization
- Bucket creation
- CRUD operations
- Metadata updates
- Concurrent access
- Error handling
- DB corruption recovery

### 6.3 GC Tests

**File:** `libbeat/statestore/backend/bbolt/gc_test.go`

Test cases:
- Disk GC removes expired entries
- Disk GC preserves active entries
- Cache GC removes expired entries (Phase 2)
- Cache GC preserves active entries (Phase 2)
- TTL calculation accuracy
- GC interval timing
- GC with concurrent operations

### 6.4 Integration Tests

Test scenarios:
- Filebeat restart with bbolt backend
- Migration from memlog to bbolt
- Large dataset handling
- Performance benchmarks
- Memory usage monitoring

## 7. Implementation Checklist

### 7.1 Phase 1: BBolt Backend (Estimated: 3-5 days)

#### Day 1-2: Core Implementation
- [ ] Create `libbeat/statestore/backend/bbolt/` directory
- [ ] Implement `registry.go`
  - [ ] Registry struct with settings
  - [ ] New() constructor with validation
  - [ ] Access() method - open/create stores
  - [ ] Close() method - cleanup
  - [ ] Basic logging
- [ ] Implement `store.go`
  - [ ] Store struct with bbolt.DB
  - [ ] openStore() - DB initialization
  - [ ] Bucket creation (data, metadata)
  - [ ] Close() method
  - [ ] Has() method
  - [ ] Get() method with access time update
  - [ ] Set() method with metadata
  - [ ] Remove() method
  - [ ] Each() method
  - [ ] SetID() method
- [ ] Implement `metadata.go`
  - [ ] metadata struct
  - [ ] updateAccessTime() helper
  - [ ] updateMetadata() helper
  - [ ] JSON serialization

#### Day 2-3: GC Implementation
- [ ] Implement `gc.go`
  - [ ] Registry-level GC goroutine
  - [ ] Store-level collectGarbage() method
  - [ ] Expired entry identification
  - [ ] Batch deletion
  - [ ] Error handling and logging
  - [ ] Graceful shutdown
- [ ] Add `doc.go` with package documentation

#### Day 3-4: Configuration & Integration
- [ ] Update `filebeat/config/config.go`
  - [ ] Add Type field
  - [ ] Add BBoltConfig struct
  - [ ] Update DefaultConfig
  - [ ] Validation logic
- [ ] Update `filebeat/beater/store.go`
  - [ ] Add backend type selection logic
  - [ ] Initialize bbolt registry
  - [ ] Handle configuration errors
  - [ ] Backward compatibility
- [ ] Add import to make bbolt available

#### Day 4-5: Testing
- [ ] Implement `bbolt_test.go`
  - [ ] Compliance tests
  - [ ] Registry tests
- [ ] Implement `store_test.go`
  - [ ] Unit tests for all Store methods
  - [ ] Concurrent access tests
  - [ ] Error cases
- [ ] Implement `gc_test.go`
  - [ ] GC functionality tests
  - [ ] TTL expiration tests
- [ ] Run full test suite
- [ ] Fix any issues
- [ ] Performance benchmarks

### 7.2 Phase 2: In-Memory Cache (Estimated: 2-3 days)

#### Day 1: Cache Implementation
- [ ] Implement `store_cache.go`
  - [ ] cacheEntry struct
  - [ ] storeWithCache struct
  - [ ] openStoreWithCache() constructor
  - [ ] Close() override
  - [ ] Get() with cache lookup
  - [ ] Set() with cache update
  - [ ] Remove() with cache invalidation
  - [ ] Has() cache-aware
  - [ ] Each() cache-aware
- [ ] Cache GC goroutine
  - [ ] runCacheGC() method
  - [ ] collectCacheGarbage() method
  - [ ] Graceful shutdown

#### Day 2: Configuration & Integration
- [ ] Update configuration (already done in Phase 1)
- [ ] Modify Registry.Access() to return cached store
- [ ] Add cache enable/disable logic
- [ ] Update documentation

#### Day 3: Testing
- [ ] Update `bbolt_test.go`
  - [ ] Compliance tests with cache
- [ ] Update `store_test.go`
  - [ ] Cache hit/miss tests
  - [ ] Cache consistency tests
- [ ] Update `gc_test.go`
  - [ ] Cache GC tests
  - [ ] Multi-layer GC tests
- [ ] Performance benchmarks
  - [ ] Cache vs no-cache comparison
  - [ ] Memory usage analysis

### 7.3 Documentation & Finalization

- [ ] Update project documentation
  - [ ] README mentions bbolt backend
  - [ ] Migration guide from memlog
  - [ ] Configuration examples
- [ ] Update CHANGELOG
- [ ] Code review
- [ ] Address feedback
- [ ] Final testing

## 8. Migration & Backward Compatibility

### 8.1 Migration Path

**Automatic migration NOT implemented** - users must manually migrate if switching backends.

**Migration steps for users:**
1. Stop Filebeat
2. Backup existing registry (memlog files)
3. Update configuration: `registry.type: bbolt`
4. Start Filebeat (bbolt starts fresh)
5. Filebeat re-processes files based on new registry

**Why no automatic migration:**
- Different data formats (JSON files vs bbolt DB)
- Complex state transformation
- Risk of data loss
- Users can keep memlog if preferred

### 8.2 Backward Compatibility

- Default remains configurable
- memlog backend still available
- Configuration changes are additive
- No breaking changes to existing configs

## 9. Performance Considerations

### 9.1 BBolt Optimizations

- `NoFreelistSync: true` - Faster writes, slight increase in DB size
- `NoGrowSync: false` - Safer default, can be tuned
- Batch operations in GC
- Read-only transactions for Get/Has/Each
- Single write transaction per operation

### 9.2 Cache Optimizations (Phase 2)

- LRU-style eviction through TTL
- No cache on Has() to avoid pollution
- Batch cache updates
- Separate mutexes for cache vs disk

### 9.3 Expected Performance

**vs memlog:**
- Faster random reads (no log replay)
- Similar write performance
- Better memory usage (with cache)
- Automatic cleanup (GC)

## 10. Error Handling

### 10.1 Error Categories

**Initialization errors:**
- DB file creation failure
- Permission errors
- Corrupt DB file

**Operation errors:**
- Transaction errors
- Encoding/decoding errors
- DB closed errors

**GC errors:**
- Non-fatal, logged only
- Retry on next interval

### 10.2 Recovery Strategies

- DB corruption: log error, attempt to continue
- Permission issues: fail fast with clear message
- GC failures: log and continue
- Graceful degradation where possible

## 11. Security Considerations

- File permissions: 0600 by default
- No sensitive data logging
- Secure defaults for bbolt options
- No network exposure

## 12. Open Questions & Decisions

### 12.1 Design Decisions Made

1. **Separate DB file per store** - Isolation, easier management
2. **JSON encoding** - Compatibility, debugging ease
3. **Two buckets** - Data/metadata separation
4. **Access time on Get** - Track usage accurately
5. **No automatic migration** - Reduced complexity, lower risk

### 12.2 Tunable Parameters

- GC intervals (= TTL values)
- BBolt options (NoFreelistSync, etc.)
- Cache enabled/disabled (Phase 2)
- File permissions

### 12.3 Future Enhancements

- Automatic migration from memlog
- Metrics/monitoring
- Compaction strategies
- Batch operation APIs
- Cache size limits (Phase 2)

## 13. Success Criteria

**Phase 1:**
- [ ] All compliance tests pass
- [ ] All unit tests pass
- [ ] GC correctly removes expired entries
- [ ] Configuration parsing works
- [ ] Backend selection works
- [ ] No memory leaks
- [ ] Performance acceptable (< 2x memlog latency)

**Phase 2:**
- [ ] Cache hit rate > 80% for typical workload
- [ ] Cache GC works correctly
- [ ] Memory usage reasonable (< 100MB for 10K entries)
- [ ] Performance improved vs Phase 1

## 14. Timeline Estimate

**Phase 1 (BBolt Backend):**
- Development: 3-5 days
- Testing: 1-2 days
- Review & fixes: 1-2 days
- **Total: 5-9 days**

**Phase 2 (In-Memory Cache):**
- Development: 2-3 days
- Testing: 1 day
- Review & fixes: 1 day
- **Total: 4-5 days**

**Overall: 9-14 days** for complete implementation.

## 15. Appendix

### 15.1 BBolt Resources

- [BBolt Documentation](https://github.com/etcd-io/bbolt)
- [BBolt Best Practices](https://github.com/etcd-io/bbolt#caveats--limitations)

### 15.2 Reference Implementations

- `libbeat/statestore/backend/memlog/` - Pattern reference
- `libbeat/statestore/backend/es/` - Simpler pattern
- BBolt examples in etcd codebase

### 15.3 Code Review Checklist

- [ ] All exported functions documented
- [ ] Error handling consistent
- [ ] Logging appropriate
- [ ] Thread-safety verified
- [ ] Resource cleanup verified
- [ ] Tests comprehensive
- [ ] No magic numbers
- [ ] Configuration validated
