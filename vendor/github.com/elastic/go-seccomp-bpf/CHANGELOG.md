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

## [1.2.0] - 2021-09-15

### Added

- Added support for arm64. [#15](https://github.com/elastic/go-seccomp-bpf/pull/15)

### Changed

- Updated syscall tables for Linux v5.14. [#16](https://github.com/elastic/go-seccomp-bpf/pull/16)

## [1.1.0] - 2019-04-10

### Added
- Added go.mod file. [#10](https://github.com/elastic/go-seccomp-bpf/pull/10)
- Added new syscalls to be in sync with Linux v5.0. [#11](https://github.com/elastic/go-seccomp-bpf/pull/11)

### Fixed
- Fixed integer overflow in BPF conditional jumps when using long lists of
  syscalls (>256). [#9](https://github.com/elastic/go-seccomp-bpf/pull/9)

## [1.0.0] - 2018-05-17

### Added
- Initial release.

[Unreleased]: https://github.com/elastic/go-seccomp-bpf/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/elastic/go-seccomp-bpf/v1.1.0...v1.2.0
[1.1.0]: https://github.com/elastic/go-seccomp-bpf/v1.0.0...v1.1.0
[1.0.0]: https://github.com/elastic/go-seccomp-bpf/v1.0.0
