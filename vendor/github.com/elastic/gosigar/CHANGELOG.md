# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

### Added

### Changed
- Changed several `OpenProcess` calls on Windows to request the lowest possible access privilege. #50

### Deprecated

### Removed

### Fixed
- Fix value of `Mem.ActualFree` and `Mem.ActualUsed` on Windows. #49
