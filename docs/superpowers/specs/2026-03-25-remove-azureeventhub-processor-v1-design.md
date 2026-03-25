# Remove azure-eventhub input processor v1

**Date:** 2026-03-25
**Issue:** https://github.com/elastic/ingest-dev/issues/5424
**Status:** Draft

## Summary

Remove the deprecated processor v1 from the `azure-eventhub` input in Filebeat. Processor v1 uses the deprecated `azure-event-hubs-go/v3` SDK which Microsoft no longer supports. Processor v2, using the modern `azeventhubs` SDK, has been the default since the v2 introduction and is a drop-in replacement.

## Motivation

- Microsoft deprecated the `azure-event-hubs-go` SDK and its core dependencies
- No further security updates for the legacy SDK
- Processor v2 is already the default and provides all v1 functionality plus additional features (credential-based auth, WebSocket transport)

## Approach

Single PR that removes v1 code, updates routing/config, and cleans up dependencies.

## Detailed Changes

### 1. Code removal

**Delete files:**
- `x-pack/filebeat/input/azureeventhub/v1_input.go`
- `x-pack/filebeat/input/azureeventhub/v1_input_test.go`
- `x-pack/filebeat/input/azureeventhub/file_persister_test.go` — imports `azure-event-hubs-go/v3/persist`, tests v1-only functionality
- `x-pack/filebeat/input/azureeventhub/tracer.go` — implements `logsOnlyTracer` for the `devigned/tab` tracing interface, used exclusively by the legacy v1 SDK; the modern `azeventhubs` SDK does not use `devigned/tab`

**Clean up dead code in `input.go`:**
- Remove the `environments` map variable (only used by `getAzureEnvironment()` in `v1_input.go`)
- Remove the `github.com/Azure/go-autorest/autorest/azure` import (only used for the `environments` map)
- Remove the `github.com/devigned/tab` import and `tab.Register()` call (v1-only tracing)

**Delete v1-specific test code** in `input_test.go`:
- `TestProcessEvents` — exercises v1 event processing
- `TestGetAzureEnvironment` — tests `getAzureEnvironment()` defined in `v1_input.go`

**Clean up v1 comment** in `client_secret.go` (line 81 reference to processor v1).

### 2. Processor version routing

**In `input.go`:**
- Remove the `switch` on `config.ProcessorVersion`
- When `processor_version` is `"v1"`, log a warning: `"processor v1 is no longer available, using v2. The processor_version option will be removed in a future release."`
- Always create a v2 input regardless of the config value
- Keep the `processorV1` constant (needed for the warning check), keep `processorV2`

**FIPS exclusion:** Keep `ExcludeFromFIPS: true` in plugin registration but add a TODO comment to investigate whether this is still needed without the deprecated SDK.

### 3. Config validation simplification

**In `config.go`:**
- Remove v1-specific validation branches (e.g., requiring `storage_account_key` alone for connection_string auth)
- Keep only v2 validation paths as the single code path
- The `processor_version` config field stays for the warning behavior but no longer influences validation
- `validateProcessorVersion()` should continue to accept `"v1"` as valid (for backwards compatibility) — it just triggers the warning path
- Keep the `processorV1` constant (used in the warning check and validation)

**Keep as-is:**
- `migrate_checkpoint` config option and default (`true`) — plan to remove in a future release alongside `processor_version`

### 4. Dependency cleanup

**Remove imports** from all files in the package:
- `github.com/Azure/azure-event-hubs-go/v3` (all sub-packages) — in `v1_input.go`, `input_test.go`, `metrics_test.go`, `file_persister_test.go`, `azureeventhub_integration_test.go`
- `github.com/Azure/azure-storage-blob-go` — in `v1_input.go`
- `github.com/Azure/go-autorest/autorest/azure` — in `input.go`, `input_test.go`
- `github.com/devigned/tab` — in `input.go`, `tracer.go`

**Run `go mod tidy`** to clean up `go.mod` and `go.sum`. Verify whether the deprecated packages survive as transitive dependencies of other Beats code — if so, document for follow-up.

### 5. Module configs and manifests

No changes needed. The 8 Azure module manifests already default to `"v2"` and the config templates reference `processor_version` via template variables which remain valid.

### 6. Testing

**Delete:**
- `file_persister_test.go` — v1-only, imports deprecated SDK
- `azureeventhub_integration_test.go` — uses deprecated SDK and stale v1 input API (`NewInput`); likely already broken. Delete and track rewrite as follow-up if integration test coverage is needed.

**Update:**
- `config_test.go` — remove test cases for v1-specific validation paths. Keep v2 validation tests and add/update test for `processor_version: v1` fallback behavior.
- `input_test.go` — remove `TestProcessEvents` and `TestGetAzureEnvironment` (both v1-specific). Remove deprecated SDK imports. Keep shared test logic.
- `metrics_test.go` — currently creates `eventHubInputV1{}` and uses `eventhub.Event` from the legacy SDK. Rewrite to use v2 types.

**Add:**
- Test that verifies: when `processor_version` is set to `"v1"`, the input creates a v2 processor and logs a warning.

### 7. Documentation

**Update `README.md`:**
- Remove/update the config example showing `processor_version: "v1"` (line ~284)
- Update the migration path reference from `v1 > v2` (line ~379)

**Update `docs/reference/filebeat/filebeat-input-azure-eventhub.md`:**
- Remove the "Connection string authentication (processor v1)" example section (lines 20-36) — this shows a `processor_version: "v1"` config
- Remove "(processor v2)" suffixes from remaining example section headings since there's only one processor now
- Update the intro paragraph (line 12) which references the legacy Event Processor Host and links to the deprecated `azure-event-hubs-go` repo
- Update `storage_account_key` description (line 264) — currently says "option is required" which was true for v1 but not for v2 with connection string or credential auth

## Future work (planned for 9.4)

- Remove the `processor_version` config field entirely
- Remove the `migrate_checkpoint` config option and migration code in `v2_migration.go`
- Investigate and resolve the `ExcludeFromFIPS` flag

## Files affected

| File | Action |
|------|--------|
| `x-pack/filebeat/input/azureeventhub/v1_input.go` | Delete |
| `x-pack/filebeat/input/azureeventhub/v1_input_test.go` | Delete |
| `x-pack/filebeat/input/azureeventhub/file_persister_test.go` | Delete |
| `x-pack/filebeat/input/azureeventhub/tracer.go` | Delete |
| `x-pack/filebeat/input/azureeventhub/azureeventhub_integration_test.go` | Delete (rewrite as follow-up) |
| `x-pack/filebeat/input/azureeventhub/input.go` | Modify — remove routing switch, `environments` map, `go-autorest`/`tab` imports, `tab.Register` call; add v1 warning |
| `x-pack/filebeat/input/azureeventhub/config.go` | Modify — simplify validation, remove v1 branches |
| `x-pack/filebeat/input/azureeventhub/config_test.go` | Modify — remove v1 test cases |
| `x-pack/filebeat/input/azureeventhub/input_test.go` | Modify — remove `TestProcessEvents`, `TestGetAzureEnvironment`, deprecated imports; add fallback test |
| `x-pack/filebeat/input/azureeventhub/metrics_test.go` | Modify — rewrite to use v2 types instead of `eventHubInputV1` |
| `x-pack/filebeat/input/azureeventhub/client_secret.go` | Modify — clean up v1 comment |
| `x-pack/filebeat/input/azureeventhub/README.md` | Modify — update v1 references |
| `docs/reference/filebeat/filebeat-input-azure-eventhub.md` | Modify — remove v1 example, update descriptions |
| `go.mod` / `go.sum` | Modify — `go mod tidy` |
