# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

- Add more build tags to `process_common.go` so the module can be used on NetBSD and OpenBSD. #24

## [0.3.0]

### Added

- Add linux Hwmon reporting interface. #16
- Add `filesystem` package. #17

### Fixed

- Fix build tags for `darwin` after refactoring of file handle metrics. #18
- Fix process filtering. #19

## [0.2.1]

### Fixed

- Fix package name for darwin in metrics setup. #15

## [0.2.0]

### Added

- Add metrics reporting setup. #14

## [0.1.0]

### Added

- First release of `github.come/elastic/elastic-agent-system-metrics`.

[Unreleased]: https://github.com/elastic/elastic-agent-system-metrics/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/elastic/elastic-agent-system-metrics/compare/v0.0.0...v0.3.0
[0.2.1]: https://github.com/elastic/elastic-agent-system-metrics/compare/v0.0.0...v0.2.1
[0.2.0]: https://github.com/elastic/elastic-agent-system-metrics/compare/v0.0.0...v0.2.0
[0.1.0]: https://github.com/elastic/elastic-agent-system-metrics/compare/v0.0.0...v0.1.0
