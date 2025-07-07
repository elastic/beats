# Filestream GZIP Support Implementation Plan

## Notes
- Feature must be opt-in via `gzip_experimental` until GA.
- Must stream-decompress; no full decompression to disk.
- Offset tracking is on decompressed bytes; require re-scanning stream to resume.
- Only fingerprint file identity allowed for GZIP; input errors otherwise.
- Treat GZIP files as non-active; append/truncate should abort and log error.
- Integrity (CRC32 & size) verified after full read; errors logged.
- Support log rotation (plain -> GZIP) via copytruncate.
- Kubernetes integration requires dual fingerprints (compressed & decompressed) and opt-in flag.
- Tech preview: emit warning.
- Exposes monitoring metric `gzip_tech_preview_enabled` when flag enabled.
- Follow Effective Go guidelines and maintain 80-char width.

#### Open questions:
- [ ] can a file me marked as "complete", a.k.a do not read it again?
  file: `input/filestream/copytruncate_prospector.go:373`
- [ ] if there is a gzip error when opening the file to create the file
descriptor (filebeat/input/filestream/fswatch.go:406), it happened in the file
scanner, the error will happen every scan as long as the file is still there.
- [ ] add GZIP support for tests on `input_integration_test.go`?

### Changes to the RFC:
 - allow data append: it might happen filebeat pickup the gzip file before it's fully written to disk.

### Milestone 1 – Configuration & Validation
- [x] Add new config flag `gzip_experimental` with validation (enforce fingerprint identity)
      files: `filebeat/input/filestream/config.go`,
      `filebeat/input/filestream/input.go`
  - [x] Add new config
  - [x] Add validation
  - [x] Emit tech-preview warning
  - [x] Register metrics tags for tech preview

### Milestone 2 – Core GZIP Reader
- [x] Implement GZIP detection by magic bytes (reader sniffing)
      files: `filebeat/input/filestream/file.go` (see `IsGZIP` and `gzipSeekerReader`)
- [x] Integrate GZIP reader
  - [x] Update `input.go` to use `File` interface and switch between `plainFile` and `gzipSeekerReader` based on GZIP detection and `gzip_experimental` flag.
  - [x] Chunked read using existing buffer_size (handled by harvester, ensure compatibility)
  - [x] Maintain decompressed offset (ensure `file.State` and logic in `input.go` handle this)
- [x] add Test integrity verification at EOF (CRC32 & ISIZE)
      files: `filebeat/input/filestream/file.go` (within `gzipSeekerReader`)
- [ ] ~Implement modification detection: abort ingestion on append/truncate during read~
      we need to read gzip files after append as filestream might start reading the file before it's fully written to disk
- [x] Instrument GZIP-specific metrics: all metrics have a GZIP version
- [x] Enhance rotation handle .gz
  - [x] Add GZIP-awareness to `onFSEvent` in `copytruncate_prospector.go`
  - [x] Add GZIP-awareness to `onFSEvent` in `prospector.go`
- [ ] save GZIP file was fully ingested so filestream won't open and seek to offset
- [ ] Ensure (test) GZIP resume logic re-reads stream from start to reach last known decompressed offset (primarily in `input.go` when handling existing state/offset for GZIP files).
- [ ] test for a corrupted file, which at the beginning some lines succeed, then
it fails.
- [ ] test for a file with multiple GZIP files
- [ ] test a gzip and plain file with the same decompressed data have the same
fingerprint
- [ ] run BenchmarkToFileDescriptor to check overhead of checking a file is GZIP

### Milestone 3 – Testing
- [ ] Add integration tests
      files: `filebeat/input/filestream/input_integration_test.go`,
      `filebeat/input/filestream/testdata/**`
  - [ ] detection
  - [ ] offset resume (basic GZIP reading and resume, e.g. in `input_test.go` or `input_integration_test.go`)
  - [ ] integrity error
  - [ ] mixed plain/GZIP, rotation, k8s scenario
  - [ ] Identify and extend existing filestream tests (e.g., in `filestream_test.go`, `prospector_test.go`, `harvester_test.go`) to cover GZIP input by parameterizing with `gzipSeekerReader` or adding GZIP-specific test cases.
  - [ ] Extend `copytruncate_prospector_test.go` to cover GZIP rotation scenarios (e.g., plain file rotates to GZIP, new GZIP file, operations on GZIP like write/truncate are ignored, GZIP rename).

### Milestone 4 – Benchmarking
- [ ] Add benchmarks using benchbuilder
      files: `filebeat/input/filestream/gzip_reader_bench_test.go`,
      `benchbuilder/**`
  - [ ] Measure performance overhead vs plain text
  - [ ] Benchmark many small GZIP files for memory/OOM risk (see fleet-server issue #2994)
  - [ ] Benchmark a huge GZIP file (>64 GiB) for memory/OOM risk
  - [ ] Benchmark Kubernetes integration with mixed plain/GZIP & rotation

### Milestone 5 – Integrations
- [ ] Update Kubernetes integration & Custom Logs config schemas
      files: `x-pack/filebeat/input/kubernetes/...`,
      `module/customlogs/config.yml`

### Milestone 6 – Documentation
- [ ] Update Filebeat docs for filestream input, add GZIP-tech-preview section
      files: `docs/filestream-gzip.asciidoc`
  - [ ] Provide example configs and usage notes

### Milestone 7 – Review & GA Prep
- [ ] PR reviews

## Current Goal
Implement and test GZIP handling in copytruncate prospector.
