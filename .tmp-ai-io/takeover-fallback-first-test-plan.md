# TakeOver Fallback - First Test Implementation Plan

## Summary
- Add a new integration test in `filebeat/tests/integration` that uses input reload (`filebeat.config.inputs`) and file output.
- Drive two `.log` files with monotonic counters using `integration.WriteLogFileFrom(...)` in small append batches so each phase has deterministic expected ranges.
- Alternate active inputs (`log` -> `filestream take_over` -> `log` -> `filestream take_over`) by creating/removing YAML files in `inputs.d`.
- After each "stop all inputs" transition, copy the current output NDJSON file to a phase snapshot file for post-phase assertions.
- Assert continuity/isolation by comparing per-input-type counter progression per source file.
- All helper functions receive `t *testing.T` and must handle failures internally (`t.Fatalf`/`t.Errorf`), returning no `error`.
- Only assertion helpers call `t.Helper()`.
- Snapshot filenames use an ever-increasing numeric prefix for sortable ordering.

## Scope and test placement
- Add test in `filebeat/tests/integration/take_over_fallback_test.go`.
- Reuse existing `integration.NewBeat`, `integration.WaitLineCountInFile`, `integration.GetEventsFromFileOutput`, `filebeat.WaitLogsContains`, and `WriteLogFileFrom`.
- Use `output.file` only (no Elasticsearch dependency).

## Test design (block-by-block)

### Block A: Start data generation on two different files
Implementation:
- Create two log file paths in `filebeat.TempDir()`, for example `log-1.log` and `log-2.log`.
- Keep per-file counters:
  - `nextCounter[file1]`, `nextCounter[file2]` initialized to `0`.
- Implement helper `appendBatch(path string, n int)`:
  - calls `nextCounter[path] = integration.WriteLogFileFrom(t, path, nextCounter[path], n, true)`.
- Seed files once with `append=true` by calling `WriteLogFileFrom(..., count=initialCount, startAt=0, append=false)` and storing returned next counters.
- During each run phase, append fixed-size batches (for example, 20 lines per file) instead of sleeping blindly; this reduces flakiness.

### Block B: Start Filebeat with no inputs in `inputs.d`
Implementation:
- Build base config with:
  - `filebeat.config.inputs.path: <tempDir>/inputs.d/*.yml`
  - `filebeat.config.inputs.reload.enabled: true`
  - `output.file.path: ${path.home}`, `output.file.filename: output-file`
  - debug logging enabled.
- Create `inputs.d` directory empty.
- Start Filebeat and wait for output file creation only after first active input is added.

### Block C: Run the log input
Implementation:
- Write `inputs.d/active.yml` with `type: log`, `allow_deprecated_use: true`, `scan_frequency: 0.1s`, and both file paths.
- Append one or more batches while log input is active.
- Wait for publication count to reach expected total.
- Capture `log` phase max counter per file from output events:
  - parse NDJSON events,
  - filter `input.type == "log"`,
  - map by `log.file.path`,
  - parse counter from event `message` (last token).

### Block D: Stop all inputs + copy output file snapshot
Implementation:
- Disable current input by renaming `inputs.d/active.yml` to `inputs.d/active.yml.disabled` (no pre-delete step; rely on `os.Rename` overwrite behavior).
- Wait for stop signal in logs (by input type):
  - `Runner: 'input [type=<input_type>]' has stopped`
- Copy current output file (`output-file-*.ndjson`) to `<NN>-output-phase-<phase>.ndjson` in temp dir, where `NN` is a strictly increasing counter (`01`, `02`, ...).
- Snapshot helper:
  - resolve single output file path with `filepath.Glob`,
  - `io.Copy` to copy the file.

### Block E: Replace Log input with Filestream take over enabled
Implementation:
- Write new `inputs.d/active.yml` with:
  - `type: filestream`
  - stable `id` (for example `take-over-from-log-input`)
  - takeover enabled (`take_over: true` or equivalent supported form)
  - do not set explicit fingerprint options; use filestream defaults
  - scanner check interval can be small (0.1s) if needed for test speed
  - same file paths.
- Append deterministic batches while filestream is active.
- Wait for expected total events in output file.
- Assert no duplicate replay at handoff:
  - first filestream counter per file must be `>=` log phase next expected counter boundary,
  - and filestream sequence itself must be strictly increasing without gaps within the generated range consumed in this phase.

### Block F: Start Log input again (fallback simulation)
Implementation:
- Disable filestream config, wait for runner stop, snapshot output.
- Re-enable log config on same files, append new batches.
- Wait for expected total.
- Assert fallback continuity:
  - new `log` events after fallback must continue from log's own previous max counter + 1 per file,
  - do not assert global dedup between log and filestream here (state isolation allows overlap across different input types).

### Block G: Start Filestream again
Implementation:
- Disable log config, wait for stop, snapshot output.
- Re-enable filestream take_over config, append batches, wait for expected total.
- Assert filestream continuity:
  - new filestream events continue from filestream's previous max counter + 1 per file,
  - no filestream-internal duplication/regression.

## Event assertion strategy
- Define local event struct in the test file with fields:
  - `Input.Type`
  - `Message`
  - `Log.File.Path`
- Parse counters from `Message` by extracting final numeric token.
- For each phase snapshot:
  - read all events,
  - partition by `(input.type, log.file.path)`,
  - derive min/max and monotonic progression checks.
- Maintain `lastSeen[inputType][path]` across phases to validate continuity exactly where required.

## Suggested helper functions for the test file
- `writeLogInputConfig(t *testing.T, inputsDir string, paths []string)`
- `writeFilestreamTakeOverConfig(t *testing.T, inputsDir string, id string, paths []string)`
- `disableActiveInput(t *testing.T, inputsDir string, filebeat *integration.BeatProc, runner string)`
- `copyOutputSnapshot(t *testing.T, tempDir string, snapshotIdx int, phase string) string`
- `readOutputEvents(t *testing.T, path string) []event`
- `counterFromMessage(t *testing.T, msg string) int`
- `assertNoDuplicationFromPreviousInput(t *testing.T, events []event, inputType string, lastSeen map[string]int)`

Helper rules:
- Error-handling helpers fail internally and do not return `error`.
- Do not use `t.Helper()` in setup/mutation/helpers that only perform I/O or state changes.
- Use `t.Helper()` only in assertion-oriented helpers (for example `assertContinuesFromLast` and any helper that validates expectations).

## Numbered incremental implementation steps
1. Create the test skeleton (`TestFilebeatTakeOverFallbackWithInputReload`) and base config template using `filebeat.config.inputs`.
2. Add setup for `inputs.d`, two log file paths, and output-file discovery helper.
3. Implement deterministic data append helper using `integration.WriteLogFileFrom` and per-file `nextCounter`.
4. Seed initial data in both files and start Filebeat with empty `inputs.d`.
5. Add helper to write active `log` input config.
6. Activate log input, append a batch, wait for expected event count.
7. Implement event parsing helpers (`readOutputEvents`, `counterFromMessage`).
8. Record baseline `lastSeen` for `input.type=log` and both files.
9. Disable all inputs, wait for runner stop, increment `snapshotIdx`, copy phase snapshot (`01-output-phase-log-1.ndjson`).
10. Add helper to write active `filestream + take_over` config.
11. Activate filestream, append batch, wait for expected total.
12. Assert filestream handoff did not re-read old log-ingested lines.
13. Update filestream `lastSeen`, then disable all inputs, increment `snapshotIdx`, and snapshot (`02-output-phase-filestream-1.ndjson`).
14. Re-activate log input, append batch, wait for expected total.
15. Assert log resumed from previous log `lastSeen` values (fallback continuity).
16. Update log `lastSeen`, disable all inputs, increment `snapshotIdx`, snapshot (`03-output-phase-log-2.ndjson`).
17. Re-activate filestream input, append batch, wait for expected total.
18. Assert filestream resumed from previous filestream `lastSeen` values.
19. Final sanity assertions:
    - both files received events in all expected phases,
    - no per-input-type counter regressions,
20. Optional cleanup/refactor pass: reduce duplicate config-writing code and centralize phase execution helper.

## Unresolved questions
- None.
