# BBolt Registry Debug Web UI Implementation Plan

## Goal
Add a simple, read-only web interface to Filebeat for debugging the bbolt registry, allowing inspection of all keys and their values with pagination.

## Architecture Overview

### Components
1. **Configuration** - Add `registry.debug_port` config option
2. **DB Access** - Expose bbolt.DB from store for read-only access
3. **HTTP Server** - Lightweight web server in Filebeat beater
4. **HTML UI** - Single-page interface with pagination
5. **Data Decoder** - Parse filestream state structures

## Detailed Implementation Steps

### Phase 1: Configuration (filebeat/config/config.go)
**File:** `filebeat/config/config.go:27-49`

Add `DebugPort` to `Registry` struct:
```go
type Registry struct {
    // ... existing fields ...
    DebugPort int `config:"debug_port"`
}
```

Update `DefaultConfig` with default port 8000:
```go
var DefaultConfig = Config{
    Registry: Registry{
        // ... existing fields ...
        DebugPort: 8000,
    },
    // ...
}
```

**Validation:** Add check in `ValidateConfig()` to ensure port is 0 (disabled) or 1024-65535.

---

### Phase 2: Expose bbolt.DB (libbeat/statestore/backend/bbolt/store.go)

**File:** `libbeat/statestore/backend/bbolt/store.go:60`

Add public getter method to `store` struct:
```go
// DB returns the underlying bbolt database for read-only access.
// Returns nil if store is closed.
func (s *store) DB() *bbolt.DB {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    if s.closed {
        return nil
    }
    return s.db
}
```

**Note:** The method uses existing RLock to ensure thread safety. No additional synchronization needed since bbolt handles concurrent readers.

---

### Phase 3: Registry DB Access (libbeat/statestore/backend/bbolt/registry.go)

**File:** `libbeat/statestore/backend/bbolt/registry.go`

Add method to retrieve DB from named store:
```go
// GetDB returns the underlying bbolt database for a named store.
// Returns nil if store doesn't exist or is closed.
// This is intended for debugging/inspection tools only.
func (r *Registry) GetDB(name string) *bbolt.DB {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if !r.active {
        return nil
    }
    
    s := r.stores[name]
    if s == nil {
        return nil
    }
    
    return s.DB()
}
```

---

### Phase 4: Debug Server Package (filebeat/beater/debug/)

Create new package: `filebeat/beater/debug/`

#### File: filebeat/beater/debug/server.go
Main server implementation with HTTP handlers.

**Key structures:**
```go
type Server struct {
    logger   *logp.Logger
    port     int
    registry *bbolt.Registry
    server   *http.Server
    storeName string
}

type PageRequest struct {
    Page     int    `json:"page"`
    PageSize int    `json:"page_size"`
    Bucket   string `json:"bucket"` // "data" or "metadata"
}

type PageResponse struct {
    Keys      []KeyValue `json:"keys"`
    Total     int        `json:"total"`
    Page      int        `json:"page"`
    PageSize  int        `json:"page_size"`
    TotalPages int       `json:"total_pages"`
    Bucket    string     `json:"bucket"`
}

type KeyValue struct {
    Key      string          `json:"key"`
    Value    json.RawMessage `json:"value"`
    Error    string          `json:"error,omitempty"`
}
```

**HTTP Endpoints:**
- `GET /` - Serve HTML UI
- `GET /api/keys` - Paginated key listing (query params: page,
  page_size). 
- `GET /api/buckets` - List available buckets

**Implementation notes:**
- Use bbolt read-only transactions for all operations
- Default page_size=50, max=1000
- Support both "data" and "metadata" buckets
- Handle pagination by iterating cursor and skipping to offset
- Return raw JSON for values (no decoding beyond JSON parsing)

#### File: filebeat/beater/debug/html.go
Embedded HTML template for the UI.

**UI Features:**
- Bucket selector (data/metadata dropdown)
- Key list with pagination controls (Previous/Next, page input)
- JSON viewer for selected key (expandable/collapsible)
- Auto-refresh option (enabled by default, every 10s)
- Search/filter by key prefix (client-side)

**Tech Stack:**
- Pure HTML/CSS/JavaScript (no external dependencies)
- JSON syntax highlighting using `<pre>` with CSS
- Responsive design for mobile/desktop
- Dark mode support (media query)

---

### Phase 5: Integration with Filebeat Beater (filebeat/beater/filebeat.go)

**File:** `filebeat/beater/filebeat.go`

Modify `Filebeat` struct to include debug server:
```go
type Filebeat struct {
    // ... existing fields ...
    debugServer *debug.Server
}
```

In `New()` function, after opening state store:
```go
// Start debug server if enabled
if config.Registry.DebugPort > 0 {
    // Extract bbolt registry from filebeatStore
    if bboltReg := extractBBoltRegistry(fb.store); bboltReg != nil {
        debugServer, err := debug.NewServer(
            fb.logger,
            config.Registry.DebugPort,
            bboltReg,
            info.Beat, // store name
        )
        if err != nil {
            fb.logger.Warnf("Failed to start debug server: %v", err)
        } else {
            fb.debugServer = debugServer
        }
    }
}
```

Add helper function to extract registry:
```go
func extractBBoltRegistry(store *filebeatStore) *bbolt.Registry {
    // Access the underlying backend.Registry
    // This requires either:
    // 1. Adding a getter to statestore.Registry, or
    // 2. Storing bbolt.Registry reference in filebeatStore during creation
    // Option 2 is cleaner for this use case
}
```

**Note:** Requires modifying `openStateStore()` in `filebeat/beater/store.go` to return/store the bbolt.Registry reference.

In `Stop()` function:
```go
if fb.debugServer != nil {
    fb.debugServer.Stop()
}
```

---

### Phase 6: Store Reference Management (filebeat/beater/store.go)

**File:** `filebeat/beater/store.go`

Modify `filebeatStore` struct:
```go
type filebeatStore struct {
    registry      *statestore.Registry
    esRegistry    *statestore.Registry
    storeName     string
    cleanInterval time.Duration
    notifier      *es.Notifier
    
    // For debug access
    bboltRegistry *bbolt.Registry
}
```

In `openStateStore()`, store the bbolt registry:
```go
switch cfg.NormalizedType() {
case "bbolt":
    reg, err = bbolt.New(logger, bbolt.Settings{...})
    if err == nil {
        store.bboltRegistry = reg.(*bbolt.Registry) // type assertion
    }
// ...
}
```

Add getter:
```go
func (s *filebeatStore) BBoltRegistry() *bbolt.Registry {
    return s.bboltRegistry
}
```

---

### Phase 7: Filestream State Decoding (filebeat/beater/debug/decoder.go)

**File:** `filebeat/beater/debug/decoder.go`

Parse filestream-specific state structures for better display.

```go
// FileStreamState matches filebeat/input/filestream/internal/input-logfile/store.go:493
type FileStreamState struct {
    ID             string        `json:"id"`
    Offset         int64         `json:"offset"`
    TTL            time.Duration `json:"ttl"`
    Source         string        `json:"source"`
    IdentifierName string        `json:"identifier_name"`
}

func DecodeFileStreamState(raw json.RawMessage) (*FileStreamState, error) {
    var state FileStreamState
    if err := json.Unmarshal(raw, &state); err != nil {
        return nil, err
    }
    return &state, nil
}
```

Enhance `KeyValue` response to include parsed state when possible:
```go
type KeyValue struct {
    Key       string           `json:"key"`
    Value     json.RawMessage  `json:"value"`
    Parsed    *FileStreamState `json:"parsed,omitempty"`
    Error     string           `json:"error,omitempty"`
}
```

---

## File Structure Summary

```
filebeat/
├── config/
│   └── config.go              # Add DebugPort field
├── beater/
│   ├── filebeat.go            # Integrate debug server
│   ├── store.go               # Store bbolt.Registry reference
│   └── debug/                 # New package
│       ├── server.go          # HTTP server and handlers
│       ├── html.go            # Embedded HTML template
│       └── decoder.go         # State structure parsing

libbeat/statestore/backend/bbolt/
├── store.go                   # Add DB() getter
└── registry.go                # Add GetDB(name) method
```

---

## Testing Strategy

### Manual Testing
Use `data/registry/filebeat.db` as test database:
1. Copy to test environment
2. Configure `registry.debug_port: 8000`
3. Start Filebeat: `./filebeat -e`
4. Open browser: `http://localhost:8000`
5. Verify key listing, pagination, JSON display
6. Test bucket switching (data/metadata)
7. Verify graceful shutdown on Ctrl+C

---

## Configuration Example

```yaml
# filebeat.yml
filebeat:
  registry:
    path: registry
    debug_port: 8000  # Enable debug UI on port 8000
                      # Set to 0 to disable (default: 8000)
```

**Security Note:** Debug server is localhost-only (bind to 127.0.0.1). Document that it should NEVER be exposed to untrusted networks.

---

## Success Criteria

1. ✓ Configuration option `registry.debug_port` works (default 8000)
2. ✓ Web UI accessible at `http://localhost:<port>`
3. ✓ Lists all keys from both "data" and "metadata" buckets
4. ✓ Pagination works (default 50 items/page)
5. ✓ JSON values displayed correctly with syntax highlighting
6. ✓ Filestream states parsed and displayed in human-readable format
7. ✓ No performance impact when debug server disabled (port=0)
8. ✓ Read-only access only (no write/delete operations)
9. ✓ Graceful shutdown on Filebeat stop
10. ✓ Thread-safe concurrent access

---

## Open Questions

1. **statestore.Registry access:** Should we add a public `Backend()` method to `libbeat/statestore/registry.go` to get the underlying `backend.Registry`? Or store it separately in `filebeatStore`?
   - **Answer:** Store separately in `filebeatStore` for cleaner separation and type safety.

2. **Port binding:** Should debug server bind to `127.0.0.1` only or allow configuration?
   - **Recommendation:** Hardcode to `0.0.0.0` for now. Can add `debug_address` config later if needed.

3. **Multiple stores:** Filebeat uses `info.Beat` as store name (typically "filebeat"). Should UI support multiple stores?
   - **Answer:** No, single store is sufficient for initial implementation. Can extend later.

4. **Metadata bucket display:** Should we decode the metadata (last_access, last_change) or show raw JSON?
   - **Recommendation:** Parse and show human-readable RFC3339

5. **UI framework:** Pure HTML/JS or use template engine?
   - **Answer:** Pure HTML/JS embedded as string constant. No external dependencies. Simpler for OSS project.

---

## Token Optimization Notes

- Store reference cheaply stored in `filebeatStore` struct (8 bytes pointer)
- DB access method uses existing locks, no additional overhead
- HTTP server runs only when `debug_port > 0`
- No goroutines spawned unless server enabled
- Read-only transactions have minimal lock contention with writers
- Pagination prevents memory spikes from large key sets

---

## Implementation Order

1. Add configuration field and validation
2. Expose DB from bbolt store/registry
3. Implement debug server package (server, handlers, HTML)
4. Integrate with filebeat beater
5. Add filestream decoder
6. Write tests
7. Manual testing with real registry
8. Documentation

Estimated time: 4-6 hours of focused development.
