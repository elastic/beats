# Filestream Harvester Progress Metrics Implementation Plan

Issue: https://github.com/elastic/beats/issues/36653

Scope: Filestream input only. Plain files only. GZIP files and files ignored for any reason are excluded.

## Target Metrics

Add three aggregate gauges for plain-file ingestion progress:

- `files_ingested_percent_100`: files where `offset == size`
- `files_ingested_percent_95_99`: files where `offset >= 95% of size && offset < size`
- `files_ingested_percent_lt_95`: files where `offset < 95% of size`

Registry path: `monitoring.metrics.filebeat.filestream`.

## Design

Use `input-logfile.Metrics` as the aggregation owner.

Rationale:

- The scanner already gets file sizes through existing `os.Lstat`/`os.Stat` calls.
- The harvester already computes the current offset while publishing events.
- Both the file watcher and harvester already receive the same `*Metrics`.
- Keeping bucket logic in `Metrics` avoids new harvester-to-watcher progress messages.
- No extra syscall is needed.

`Metrics` should maintain an in-memory per-source progress table:

```go
type fileProgress struct {
    size   int64
    offset int64
    bucket progressBucket
    knownSize bool
    knownOffset bool
}
```

Only count an entry once both size and offset are known, size is positive, the file is not GZIP, and the file is not ignored.

## Code Changes

1. Extend `input-logfile.Metrics`.
   - Add three gauge fields.
   - Add a mutex-protected `map[string]fileProgress`.
   - Add helper methods:
     - `UpdateFileSize(id string, size int64, gzip bool, ignored bool)`
     - `UpdateFileOffset(id string, offset int64)`
     - `RemoveFileProgress(id string)`
     - `CleanupFileProgressMetrics()`
   - Recompute only the affected source on each update:
     - decrement old bucket if counted
     - update source state
     - increment new bucket if counted

2. Update file watcher scan path.
   - In `fileWatcher.watch`, after calculating `srcID`, call `metrics.UpdateFileSize(srcID, fd.Info.Size(), fd.GZIP, ignored)` for current files.
   - Reuse the existing ignore logic from `fileIgnoreReason`; do not count files ignored by `ignore_older`, `ignore_inactive`, include/exclude filters, fingerprint-too-small handling, empty-file handling, or any other scanner/prospector ignore path.
   - Do not register GZIP or ignored files.
   - Call `metrics.RemoveFileProgress(srcID)` when a file is removed.
   - On rename, preserve progress if the source ID is unchanged; otherwise remove the old ID and let the new descriptor register naturally.

3. Update harvester read path.
   - In `filestream.readFromSource`, after `s.Offset` advances, call `metrics.UpdateFileOffset(sourceID, s.Offset)`.
   - Pass the source ID into `readFromSource`, probably from `src.Name()` or from the outer `Run` context.
   - Do not update progress for GZIP. Either pass `isGZIP` into the helper and let it ignore, or avoid the call when `isGZIP`.
   - When the harvester closes, call `metrics.RemoveFileProgress(sourceID)` so closed files no longer contribute to the progress gauges.

4. Handle initial offset state.
   - After `initState`, before reading starts, register the current offset with metrics for non-GZIP files.
   - This ensures restarted harvesters and already-partially-ingested files are counted before the next message is read.

5. Handle truncation and ignored files.
   - Truncation already resets cursor to zero. Ensure progress offset is updated to zero for the affected source.
   - Ignored files must be removed from progress tracking, not counted as complete, even when the cursor is reset to file size.

6. Cleanup on input shutdown.
   - In file watcher shutdown, call progress cleanup alongside `CleanupFileScanMetrics`.
   - Ensure cleanup subtracts this input's current bucket contributions from shared aggregate gauges.

## Bucket Calculation

Avoid float math:

```go
switch {
case offset >= size:
    bucket = bucketComplete
case offset*100 >= size*95:
    bucket = bucket95To99
default:
    bucket = bucketBelow95
}
```

Guard against overflow if using multiplication. File sizes are `int64`; use division-based comparison or `math/bits` only if needed. A simple safer comparison is:

```go
case offset >= size:
case offset >= (size+19)/20*19:
```

But that expression rounds in a way that needs careful tests. Prefer a small helper with explicit test cases around boundaries.

## Tests

Add unit tests for `Metrics`:

- unknown size does not count
- unknown offset does not count
- size zero does not count
- GZIP is excluded
- ignored file is excluded
- below 95 bucket
- exactly 95 bucket
- 99 bucket
- exactly complete bucket
- offset greater than size counts complete
- source moving between buckets updates aggregate gauges by delta
- remove source decrements previous bucket
- cleanup removes all current contributions

Add focused watcher/harvester tests where practical:

- watcher reports size without extra filesystem calls beyond scan
- harvester reports offset updates for plain files
- GZIP file does not update progress buckets
- ignored file does not update progress buckets
- harvester close clears progress state

## Validation

Run scoped tests only:

```bash
cd filebeat
go test -v ./input/filestream/internal/input-logfile/... ./input/filestream/...
```

If full Filestream tests are too broad or slow, run the specific packages/tests touched first.

## Risks

- Shared global registry gauges require correct cleanup, otherwise stopped inputs can leave stale counts.
- Rename and copy-truncate flows can double-count if old source IDs are not removed correctly.
- Boundary math around 95% must be exact and integer-only.
- GZIP exclusion must be explicit so compressed files do not skew plain-file progress counts.
- Ignored-file exclusion must be applied consistently across scanner and prospector ignore paths.
