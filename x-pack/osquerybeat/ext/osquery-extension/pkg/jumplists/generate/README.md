# Jumplists Generator

This generator produces:

- `../generated_app_ids.go`
- `../generated_guid_mappings.go`

## Source Cache

Upstream source files are cached locally for deterministic generation:

- `sources/AppIDs.txt`
- `sources/GuidToName.txt`

By default, generation reads from these cached files and does not download.

## Refreshing Sources On Demand

To refresh cached source files from upstream, run:

```bash
cd x-pack/osquerybeat/ext/osquery-extension/pkg/jumplists
go run ./generate -refresh-sources
```

This updates the files in `generate/sources/` and then regenerates the Go outputs.

## Mage Integration

From `x-pack/osquerybeat`:

- `mage generate` uses cached source files by default.
- `JUMPLISTS_REFRESH_SOURCES=true mage generate` refreshes cached source files first.
