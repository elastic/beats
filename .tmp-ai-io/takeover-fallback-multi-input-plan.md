# Take Over Fallback Multi-Input Plan

## Summary
- Extend the existing fallback test to run with multiple input pairs, where each pair (`log` + `filestream`) owns a disjoint file set.
- Keep assertions based only on `input.type` + `log.file.path` (no per-input-id assertions needed).
- Use file names encoding group membership, e.g. `group-01-file-A.log`, `group-01-file-B.log`, `group-02-file-A.log`, `group-02-file-B.log`.
- Keep templates simple and hard-coded per mode: one template file for multi-log mode and one for multi-filestream mode, switched by replacing `inputs.d/active.yml`.
- Reuse existing assertion helpers and generalize only where needed to iterate all files/groups.

## Scope and assumptions
- Number of groups: start with `2` (`group-01`, `group-02`) to prove isolation between pairs.
- Two files per group (`file-A`, `file-B`).
- `log` template defines one `log` input per group (IDs not required).
- `filestream` template defines one `filestream` input per group with IDs matching the group (`group-01`, `group-02`), and takeover enabled.
- Paths are disjoint by group glob (e.g. `.../group-01-*.log`, `.../group-02-*.log`) to avoid intentional overlap.

## Template design

### `testdata/take-over-fallback/log-input.yml`
- Hard-code two `log` inputs:
  - input 1 paths: `{{.logDir}}/group-01-*.log`
  - input 2 paths: `{{.logDir}}/group-02-*.log`
- Keep `allow_deprecated_use`, scan interval, etc, aligned with current test behavior.

### `testdata/take-over-fallback/filestream-input.yml`
- Hard-code two `filestream` inputs:
  - `id: group-01`, paths `{{.logDir}}/group-01-*.log`
  - `id: group-02`, paths `{{.logDir}}/group-02-*.log`
- `take_over` enabled on both inputs.
- No explicit fingerprint tuning; keep defaults.

## Test structure changes

### 1) File layout and generation
- Replace current `log-1.log` / `log-2.log` with:
  - `group-01-file-A.log`
  - `group-01-file-B.log`
  - `group-02-file-A.log`
  - `group-02-file-B.log`
- Build `logFiles []string` from those names.
- Keep current `nextCounter[path]` map and append helper unchanged in principle.

### 2) Config switching
- Keep `base.yml` unchanged.
- Keep existing template names and update their content for multi-input mode:
  - `writeLogInputConfig` continues rendering `log-input.yml`
  - `writeFilestreamConfig` continues rendering `filestream-input.yml`
- Continue using `inputs.d/active.yml` rename strategy to disable.

### 3) Assertions
- Continue using existing helpers:
  - `countExtremesByPath`
  - `assertNoDuplicationFromPreviousInput`
  - `assertContinuesFromLast`
  - `assertPerInputCountersStrictlyIncrease`
- Because each group has dedicated files, path-level assertions naturally validate pair isolation.
### 4) Event accounting
- Expected event counting remains the same model:
  - compute deltas from `nextCounter[path]-1` vs `lastSeen[inputType][path]`.
- This scales linearly with additional files/groups and does not require per-input-id tracking.

## Detailed incremental implementation plan

1. Add multi-input templates:
   - update existing `log-input.yml`
   - update existing `filestream-input.yml`
2. Update test file naming to group-based names and build `logFiles`.
3. Add `logDir` template variable and switch `write*Config` helpers to multi templates.
4. Keep phase flow unchanged (log -> filestream -> log -> filestream).
5. Re-run phase 1 baseline capture; verify all 4 files ingested by `log`.
6. Re-run phase 2 takeover assertion; verify `filestream` starts strictly after `log` boundaries for all 4 files.
7. Re-run fallback phase (`log` re-enabled); verify continuity from previous `log` boundaries for all 4 files.
8. Re-run final filestream phase; verify continuity from previous `filestream` boundaries for all 4 files.
9. Add/enable final sanity checks:
   - strict monotonicity by `(input.type, path)`
   - every file seen by both input types across phases
   - snapshot files present and ordered
10. Optional cleanup:
   - extract file-name construction into helper (`buildGroupFiles(groups []string)`),
   - extract startup waits into phase helper if repetition grows.

## Validation strategy
- Run only the targeted integration test:
  - `go test -tags integration ./filebeat/tests/integration -run TestFilebeatTakeOverFallbackWithInputReload -count=1`
- Run it repeatedly (at least 3x) to detect flakiness from reload timing.

## Risks and mitigations
- Reload race when switching templates:
  - keep explicit `WaitLogsContains(...)` for runner startup/stop.
- False positives from stale output reads:
  - keep phase deltas and slice-based assertions (`events[prevExpectedEvents:]`).
- Hidden overlap in file globs:
  - use strict `group-XX-*.log` patterns and assert expected file ownership.
