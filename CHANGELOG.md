# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).


## [Unreleased]

### Added

- Add metrics from `/proc/[pid]/io`
- Add metrics for disk reads and writes per cgroup

### Changed

### Deprecated

### Removed

### Fixed

 - Fix CmdLine generation and caching for system.process
 - Fix process metrics collection when both cgroup V1 and V2 controllers exist
 - Skip permissions errors when reading /proc/pid/io
 - Skip non-fatal errors in cgroup data collection #134

## [0.7.0]

### Added

- Collect the number of threads as part of the information collected per process
on `metric/system/process.GetInfoForPid`.

### Fixed

## [0.6.1]

### Changed

-Bump up go-sysinfo dependency version #86

## [0.6.0]

### Added

- Add user data to monitoring setup #80

### Changed
- Update host functions with `go-sysinfo` API changes #81
- Move SetupInfoUserMetrics out of a cgo build constraint #82
- Bumping up version for go-sysinfo dependency #84

### Fixed

-Ensure correct devID width in cgv2 #74

## [0.4.6]

### Fixed

- Fix type issues for MIPS platforms
- metric/system/cgroup/cgv2: ensure Rdev is correct width

## [0.4.5]

### Added

- add network data to process metrics
- add Process State API

### Changed
- Go version 1.18.4
- move exported helpers to _common.go file

### Removed

- remove hostfs checks

### Fixed

- fix(process): Typo in getargs error message
- fix(process): Make process package buildable on AIX

## [0.4.4]

### Fixed

- Fix thread safety in process code #43

## [0.4.3]

## Fixed

- Add IsZero to CPU Totals, report value for monitoring #40

## [0.4.2]

## Fixed

- Fix reporting "handles" metrics as part of the "system" metrics group instead of the "beat" process metrics group. #37

## [0.4.1]

### Added

- Add `D` process states. #32
- Move `cpu` from Metricbeat to support CPU info on Linux. #36

## [0.4.0]

### Added

- Move packages from Metricbeat: `internal/metrics/cpu` and `internal/metrics/memory`. #27

## [0.3.1]

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

[Unreleased]: https://github.com/elastic/elastic-agent-system-metrics/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/elastic/elastic-agent-system-metrics/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/elastic/elastic-agent-system-metrics/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/elastic/elastic-agent-system-metrics/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/elastic/elastic-agent-system-metrics/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/elastic/elastic-agent-system-metrics/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/elastic/elastic-agent-system-metrics/compare/v0.0.0...v0.1.0
