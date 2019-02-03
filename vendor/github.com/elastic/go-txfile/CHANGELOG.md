# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

## [0.0.6]

### Fixed
- Fix flush callback not being executed on success. PR #34

## [0.0.5]

### Fixed
- Panic on atomic operation (arm, x86-32) and File lock not released when panic occurs. PR #31

## [0.0.4]

### Added
- Added `Observer` to txfile for collecting per transaction metrics. PR #23
- Make file syncing configurable. PR #29
- Added `Observer` to pq package for collecting operational metrics. PR #26

### Changed
- Queue reader requires explicit transaction start/stop calls. PR #27 

## [0.0.3]

### Fixed
- Fix build for *BSD. PR #20


## [0.0.2]

### Added
- Add `(*pq.Reader).Begin/Done` to reuse a read transaction for multiple reads. PR #4
- Add `Flags` to txfile.Options. PR #5
- Add support to increase a file's maxSize on open. PR #5
- Add support to reduce the maximum file size PR #8
- Add support to pre-allocate the meta area. PR #7
- Improved error handling and error reporting. PR #15, #16, #17, #18
- Begin returns an error if transaction is not compatible to file open mode. PR #17
- Introduce Error type to txfile and pq package. PR #17, #18

### Changed
- Refine platform dependent file syncing. PR #10
- Begin methods can return an error. PR #17

### Fixed
- Windows Fix: Add missing file unlock on close, so file can be reopened and locked. PR #11
- Windows Fix: Can not open file because '<filename>' can not be locked right now. PR #11
- Windows Fix: Max mmaped area must not exceed actual file size on windows. PR #11


[Unreleased]: https://github.com/elastic/go-txfile/compare/v0.0.6...HEAD
[0.0.6]: https://github.com/elastic/go-txfile/compare/v0.0.5...v0.0.6
[0.0.5]: https://github.com/elastic/go-txfile/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/elastic/go-txfile/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/elastic/go-txfile/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/elastic/go-txfile/compare/v0.0.1...v0.0.2
