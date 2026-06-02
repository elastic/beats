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

Use `input-logfile.Metrics` as the aggregation owner, but keep the harvester hot path lock-free.

Rationale:

- The scanner already gets file sizes through existing `os.Lstat`/`os.Stat` calls.
- The harvester already computes the current offset while publishing events.
- Both the file watcher and harvester already receive the same `*Metrics`.
- Keeping bucket logic in `Metrics` avoids new harvester-to-watcher progress messages.
- No extra syscall is needed.

`Metrics` should maintain an in-memory per-source active offset table:

```go
map[string]*atomic.Int64
```

The harvester receives a pointer to its active offset when it starts and only stores the latest offset atomically while reading. The file watcher computes bucket counts on scan using:

- current file size from `FileDescriptor.Info.Size()`
- ignored/GZIP status from the scan/prospector logic
- latest active harvester offset loaded from the `*atomic.Int64`

Only count a source when it has an active offset entry, size is positive, the file is not GZIP, and the file is not ignored. Files are removed from the active table when the harvester closes.

## Benefits

- No global mutex in the harvester read loop.
- Hot path cost is one atomic offset store per message, similar in cost profile to existing atomic metrics.
- Bucket calculation is moved to the file watcher scan cadence, where fresh file size is already available.
- No extra `stat` syscall is needed.
- Progress percentages are scan-consistent: size is the latest scanner-observed size, which matches Filestream's existing file-change model.
- Shared gauge updates happen once per scan by delta, not once per harvested line.

## Code Changes

1. Extend `input-logfile.Metrics`.
   - Add three gauge fields.
   - Add a mutex-protected `map[string]*atomic.Int64` for active plain-file offsets.
   - Add a `lastHarvesterMetrics` snapshot so shared gauges can be updated by delta, like `UpdateFileScanMetrics`.
   - Add helper methods:
     - `RegisterHarvesterOffset(id string, offset int64) *atomic.Int64`
     - `UpdateHarvesterBuckets(current HarvesterMetrics)`
     - `RemoveHarvesterOffset(id string)`
     - `CleanupHarvesterMetrics()`
   - Keep map mutation behind a mutex, but do not acquire that mutex from the harvester read loop.

2. Add active offset API.
   - `RegisterHarvesterOffset` stores and returns a `*atomic.Int64`.
   - The harvester keeps the returned offset pointer in a local variable for the lifetime of the run.
   - The read loop updates it directly with `offset.Store(s.Offset)`.
   - The watcher scan reads it directly with `offset.Load()`.

3. Update file watcher scan path.
   - In `fileWatcher.watch`, build a `HarvesterMetrics` snapshot during the existing scan.
   - For each current file, after calculating `srcID`, look up the active offset.
   - If there is no active offset, do not count the file.
   - If the file is GZIP, ignored, or size <= 0, do not count the file.
   - Load the latest offset from the `*atomic.Int64` and classify it against `fd.Info.Size()`.
   - Call `metrics.UpdateHarvesterBuckets(snapshot)` once per scan.
   - Reuse the existing ignore logic from `fileIgnoreReason`; do not count files ignored by `ignore_older`, `ignore_inactive`, include/exclude filters, fingerprint-too-small handling, empty-file handling, or any other scanner/prospector ignore path.
   - On rename, the active offset remains keyed by source ID. If the source ID changes, the old harvester should close/remove its offset entry and the new harvester should register a new one.

4. Update harvester path.
   - After `initState`, register progress for non-GZIP files using the current state offset.
   - Pass the source ID into `readFromSource`, probably from `src.Name()` or from the outer `Run` context.
   - In `filestream.readFromSource`, after `s.Offset` advances, call `activeOffset.Store(s.Offset)` if an active offset exists.
   - Do not register or update progress for GZIP files.
   - When the harvester closes, call `metrics.RemoveHarvesterOffset(sourceID)` so closed files no longer contribute to the progress gauges.

5. Handle initial offset state.
   - After `initState`, before reading starts, register the current offset with metrics for non-GZIP files.
   - This ensures restarted harvesters and already-partially-ingested files are counted before the next message is read.
   - The file is only counted after the watcher scan observes its size and sees it is not ignored.

6. Handle truncation and ignored files.
   - Truncation already resets cursor to zero. Ensure progress offset is updated to zero for the affected source.
   - Ignored files must be removed from progress tracking, not counted as complete, even when the cursor is reset to file size.
   - If a file becomes ignored while a harvester is still active, the watcher scan must exclude it from the bucket snapshot.

7. Cleanup on input shutdown.
   - In file watcher shutdown, call harvester metrics cleanup alongside `CleanupFileScanMetrics`.
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
- inactive/no-harvester file is excluded
- below 95 bucket
- exactly 95 bucket
- 99 bucket
- exactly complete bucket
- offset greater than size counts complete
- source moving between buckets across scans updates aggregate gauges by delta
- remove source decrements previous bucket
- cleanup removes all current contributions
- harvester offset store does not acquire the metrics map mutex

Add focused watcher/harvester tests where practical:

- watcher reports size without extra filesystem calls beyond scan
- harvester stores offset updates in its active offset for plain files
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
- Progress percentages are based on the latest scanner-observed file size. In append-heavy workloads, the metric can lag behind real file growth until the next scan.

## Commit Message

```text
Add Filestream harvester ingestion progress metrics

Track active harvesters by ingestion percentage so users can see which files are fully caught up, nearly caught up, or lagging. Use scanner-observed file sizes and atomic harvester offsets to avoid extra syscalls and keep the harvester read path free of shared mutex contention.

GZIP files and files ignored for any reason are excluded because their progress cannot be represented accurately by plain-file offset/size comparisons.

GenAI-Assisted: Yes
Human-Reviewed: Yes
Tool: Cursor, Model: GPT-5.5 Agent Mode
Assisted-By: Cursor
```
