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

## Milestones

### Milestone 0 – Analysis
- [ ] Analyse existing filestream harvester code paths for decompression insertion point  
      files: `filebeat/input/filestream/internal/input-logfile/harvester.go`, 
      `filebeat/input/filestream/internal/input-logfile/prospector.go`, 
      `filebeat/input/filestream/prospector.go`

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
- [ ] Integrate streaming GZIP reader into harvester
      Relevant files: `filebeat/input/filestream/input.go` (primary integration point),
      `filebeat/input/filestream/file.go` (provides `File` interface, `plainFile`, `gzipSeekerReader`),
      `filebeat/input/filestream/internal/input-logfile/harvester.go` (consumes the reader),
      `filebeat/input/file/state.go` (for offset management)
  - [x] Update `input.go` to use `File` interface and switch between `plainFile` and `gzipSeekerReader` based on GZIP detection and `gzip_experimental` flag.
  - [x] Chunked read using existing buffer_size (handled by harvester, ensure compatibility)
  - [x] Maintain decompressed offset (ensure `file.State` and logic in `input.go` handle this)
  - [ ] Ensure GZIP resume logic re-reads stream from start to reach last known decompressed offset (primarily in `input.go` when handling existing state/offset for GZIP files). This needs review after recent changes to truncation logic in `input.go`.
- [ ] Implement integrity verification at EOF (CRC32 & ISIZE)  
      files: `filebeat/input/filestream/file.go` (within `gzipSeekerReader`)
- [ ] Implement modification detection: abort ingestion on append/truncate during read  
      files: `filebeat/input/filestream/internal/input-logfile/harvester.go`,
      `filebeat/input/file/state.go`
- [ ] Instrument GZIP-specific metrics (`gzip_validation_errors_total`, `gzip_bytes_compressed_total`, `gzip_bytes_decompressed_total`)  
      files: `filebeat/input/filestream/internal/input-logfile/metrics.go`,
      `filebeat/input/filestream/input.go`
- [ ] Enhance copytruncate rotation path to handle .gz  
      files: `filebeat/input/filestream/copytruncate_prospector.go`,
      `filebeat/input/filestream/internal/input-logfile/manager.go`
  - [ ] Add GZIP-awareness to `onFSEvent` in `copytruncate_prospector.go` (user has started this).
  - [ ] Implement logic in `isRotated` and `onRotatedFile` to correctly handle GZIP files and plain-to-GZIP rotations.

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