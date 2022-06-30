# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

## [0.2.8]

### Changed

- Fix bug introduced by mapstr optimization: #65

## [0.2.7]

### Changed

- mapstr: optimyze Clone #64

## [0.2.6]

### Added

### Changed

- Modify npipe package to directly get the user SID. #62

### Deprecated

### Removed

### Fixed

- Remove VerificationMode option to empty string. Default will be `full` #59

## [0.2.5]

### Changed

- Upgrade the YAML package dependency: #56

## [0.2.4]

### Fixed

- Include group write permission in runtime directories. #49
- Fix keystore secrets parsing when values contain commas. #50

## [0.2.3]

### Added

- Add flatten keys functionality to `mapstr`. #45

## [0.2.2]

### Added

- Extracted `dialers` helpers from Metricbeat. #44

## [0.2.0]

### Added

- Pick changes from `tlscommon`. #41
- Extracted `monitoring/report/buffer`. #42

## [0.1.2]

### Added

- Extracted `kibana.Client`. #32
- Extracted `opt` and `transform`. #33
- Extracted `match` and `datetime`. #32
- Port kibana and apm changes. #35
-  Extract function `SyncParent` to reuse in elastic agent. #36

## [0.1.1]

### Fixed

- Make linting compatible with older Git. #22

## [0.1.0]

### Added

- Extracted `common.Config`. #3
- Extracted `common.MapStr`. #13
- Extracted `cloudid`. #14
- Extracted `atomic`. #15
- Extracted `api`, `api/npipe` and `monitoring`. #17
- Moved `cfgwarn` to `logp/cfgwarn`. #18
- Extracted `keystore`. #20
- Extracted `tlscommon`. #19
- Extracted `transport` and `testing`. #21
- Extracted `service`. #22

[Unreleased]: https://github.com/elastic/elastic-agent-libs/compare/v0.2.4...HEAD
[0.2.4]: https://github.com/elastic/elastic-agent-libs/compare/v0.2.3...v0.2.4
[0.2.3]: https://github.com/elastic/elastic-agent-libs/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/elastic/elastic-agent-libs/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/elastic/elastic-agent-libs/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/elastic/elastic-agent-libs/compare/v0.1.2...v0.2.0
[0.1.2]: https://github.com/elastic/elastic-agent-libs/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/elastic/elastic-agent-libs/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/elastic/elastic-agent-libs/compare/v0.0.0...v0.1.0
