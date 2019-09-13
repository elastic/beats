# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

### Security

## [1.1.0] - 2019-08-22

### Added

- Add `VMStat` interface for Linux. [#59](https://github.com/elastic/go-sysinfo/pull/59)

## [1.0.2] - 2019-07-09

### Fixed

- Fixed a leak when calling the CommandLineToArgv function. [#51](https://github.com/elastic/go-sysinfo/pull/51)
- Fixed a crash when calling the CommandLineToArgv function. [#58](https://github.com/elastic/go-sysinfo/pull/58)

## [1.0.1] - 2019-05-08

### Fixed

- Add support for new prometheus/procfs API. [#49](https://github.com/elastic/go-sysinfo/pull/49)

## [1.0.0] - 2019-05-03

### Added

- Add Windows provider implementation. [#22](https://github.com/elastic/go-sysinfo/pull/22)
- Add Windows process provider. [#26](https://github.com/elastic/go-sysinfo/pull/26)
- Add `OpenHandleEnumerator` and `OpenHandleCount` and implement these for Windows. [#27](https://github.com/elastic/go-sysinfo/pull/27)
- Add user info to Process. [#34](https://github.com/elastic/go-sysinfo/pull/34)
- Implement `Processes` for Darwin. [#35](https://github.com/elastic/go-sysinfo/pull/35)
- Add `Parent()` to `Process`. [#46](https://github.com/elastic/go-sysinfo/pull/46)

### Fixed

- Fix Windows registry handle leak. [#33](https://github.com/elastic/go-sysinfo/pull/33)
- Fix Linux host ID by search for older locations for the machine-id file. [#44](https://github.com/elastic/go-sysinfo/pull/44)

### Changed

- Changed the host containerized check to reduce false positives. [#42](https://github.com/elastic/go-sysinfo/pull/42) [#43](https://github.com/elastic/go-sysinfo/pull/43)

[Unreleased]: https://github.com/elastic/go-sysinfo/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/elastic/go-sysinfo/releases/tag/v1.1.0
[1.0.2]: https://github.com/elastic/go-sysinfo/releases/tag/v1.0.2
[1.0.1]: https://github.com/elastic/go-sysinfo/releases/tag/v1.0.1
[1.0.0]: https://github.com/elastic/go-sysinfo/releases/tag/v1.0.0
