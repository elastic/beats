# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

## [0.0.2]

### Added

- Add MultiErrGroup (#20).
- Add Group interface and TaskGroup implementation (#23).
- Add SafeWaitGroup (#23).
- Add ClosedGroup (#24).

### Changed

- FromCancel returns original context.Context, if input implements this type. Deadline and Value will not be ignored anymore. (#22)


[Unreleased]: https://github.com/elastic/go-ucfg/compare/v0.0.2...HEAD
[0.0.2]: https://github.com/elastic/go-ucfg/compare/v0.0.1...v0.0.2
