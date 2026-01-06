## Summary
- Add new `bbolt` statestore backend under `libbeat/statestore/backend/bbolt/` (registry + store + metadata + full-scan TTL GC).
- Make **bbolt the default Filebeat registry backend** via `filebeat.registry.type` (still supports `memlog`).
- Add a Filebeat integration test to ensure **registry state is persisted to bbolt on shutdown** (even with `flush: 24h`).
- Update OSS + x-pack Filebeat tests/templates that assume `registry/filebeat/log.json` to explicitly use `memlog`.

## Details
- **New backend package**: `libbeat/statestore/backend/bbolt/`
  - `registry.go`: registry implementation, per-store DB files, background disk GC ticker (interval = `disk_ttl`).
  - `store.go`: `backend.Store` implementation (CRUD + `Each`), JSON encoding, metadata updates on Get/Set, `slices.Clone` for bbolt value copies, injectable clock (`now func() time.Time`).
  - `gc.go`: Phase 1/2 full-scan GC based on `metadata.last_access` vs `disk_ttl`, logs duration/scanned/deleted.
  - `doc.go`, `error.go`.
  - Debug helpers (read-only accessors) added in later edits: `Registry.GetDB(...)`, `store.DB()`.
- **Filebeat config**: `filebeat/config/config.go`
  - `filebeat.registry.type` (default **`bbolt`**).
  - `filebeat.registry.bbolt.*` settings (disk TTL, timeout, bbolt options, etc.).
  - `Registry.ValidateConfig()` used by the beater to validate `type` and config values without triggering validation during `Unpack`.
  - Debug config added in later edits: `filebeat.registry.debug_port` (default 8000).
- **Backend selection**: `filebeat/beater/store.go`
  - Select between `bbolt` (default) and `memlog`.
  - Debug accessors added in later edits: keep `*bbolt.Registry` on `filebeatStore` + `BBoltRegistry()` getter.
- **Debug webserver (note)**: `filebeat.registry.debug_port` was introduced (default `8000`) to enable a read-only web interface for inspecting the registry during development/troubleshooting. Intended for local/debug use only.
- **Tests**
  - New test: `filebeat/tests/integration/bbolt_registry_shutdown_test.go` verifies bbolt DB contains filestream state after shutdown.
  - Existing tests/templates that rely on memlog’s `registry/filebeat/log.json` are pinned to `memlog`:
    - OSS: `filebeat/tests/integration/*`, `filebeat/testing/integration/*`, `filebeat/tests/system/config/*.yml.j2`, etc.
    - x-pack: `x-pack/filebeat/tests/system/config/filebeat_modules.yml.j2`, `x-pack/filebeat/tests/integration/registrydiagnostics_test.go`, `x-pack/filebeat/fbreceiver/receiver_test.go`.

## Test plan
- Unit tests:
  - `go test ./libbeat/statestore/backend/...`
  - `go test ./filebeat/config`
  - `go test ./filebeat/beater`
- Build:
  - `go build .` (from `filebeat/`)
- Integration (requires `filebeat/filebeat.test`):
  - `go test -c ./filebeat -o filebeat/filebeat.test`
  - `go test -tags=integration ./filebeat/tests/integration -run TestBBoltRegistrySyncedOnShutdown -count=1`

