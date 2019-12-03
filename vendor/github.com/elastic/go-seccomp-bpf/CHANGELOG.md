# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2019-04-10

### Added
- Added go.mod file. #10
- Added new syscalls to be in sync with Linux v5.0. #11

### Fixed
- Fixed integer overflow in BPF conditional jumps when using long lists of
  syscalls (>256). #9

## [1.0.0] - 2018-05-17

### Added
- Initial release.

[Unreleased]: https://github.com/olivierlacan/keep-a-changelog/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/elastic/go-seccomp-bpf/v1.0.0...v1.1.0
[1.0.0]: https://github.com/elastic/go-seccomp-bpf/v1.0.0
