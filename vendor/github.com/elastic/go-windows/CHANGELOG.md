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

## [1.0.1] - 2019-08-28

### Security

- Load DLLs only from Windows system directory.

## [1.0.0] - 2019-04-26

### Added

- Add GetProcessMemoryInfo. #2
- Add APIs to fetch process information #6.
  - NtQueryInformationProcess
  - ReadProcessMemory
  - GetProcessImageFileName
  - EnumProcesses
- Add GetProcessHandleCount to kernel32. #7

[Unreleased]: https://github.com/elastic/go-windows/compare/v1.0.1...HEAD
[1.0.1]: https://github.com/elastic/go-windows/v1.0.1
[1.0.0]: https://github.com/elastic/go-windows/v1.0.0
