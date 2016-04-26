# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

## [0.2.0]

### Added
- Support for validation via Validator interface. #16
- Added direct support for uint values. #8, #16
- Support for simple validators via struct tags (e.g. min, max, nonzero, required). #16
- Add support for validating time.Duration. #9, #16
- Added Unpacker interface for customer unpackers. #17
- Support for numeric indices for accessing/writing array elements. #12 #19

### Changed
- Set/Get methods require index of -1 if value is not supposed to be in an array. #19
- Configurations can be arrays and/or objects at the same time. #19
- Access elements with empty path and index in array based Configuration nodes. #19

### Fixed
- Check for integer overflow when unpacking into int/uint. #8, #16

## [0.1.1]

### Fixed
- Fixed unpacking *regexp.Regexp
- Fixed unpacking empty config as *Config object

## [0.1.0]

### Added
- add support for unpacking *regexp.Regexp via regexp.Compile
- Parse time.Duration from int/float values in seconds
- Improve error messages
- Add options and PathSep support to low level option setters/getters
- Added support for _rebranding_ `*ucfg.Config` via `type MyConfig ucfg.Config` using
  casts between pointer types in Unpack and Merge.
- Introduced CHANGELOG.md for documenting changes to ucfg.


[Unreleased]: https://github.com/urso/ucfg/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/urso/ucfg/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/urso/ucfg/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/urso/ucfg/compare/v0.0.0...v0.1.0
