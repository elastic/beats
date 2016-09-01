# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

## [0.3.4]

### Fixed
- Fix error message if Unpack returns error. #56

## [0.3.3]

### Fixed
- Fix `(*FlagValue).String` panic with go 1.7 #54

## [0.3.2]

### Changed
- Turn '$' into universal escape character, so '}' in default values can be escaped with '$'. #52

### Fixed
- Fix parsing ':' in expansion default value. #51, #52

## [0.3.1]

### Added
- Add `(*Config).IsArray` and `(*Config).IsDict`. #44

### Fixed
- Fix (*Config).CountField returning 1 for arrays of any size. #43
- Fix unpacking into slice/array top-level or if `inline`-tag is used. #45

## [0.3.0]

### Added
- Added CLI flag support. #15
- Added variable expansion support. #14

### Changed
- Report error message from regexp.Compile if compilation fails #21

### Fixed
- Nil values become merge-able with concrete types. #26
- Fix merging types `time.Duration` and `*regexp.Regexp`. #25
- Fix Validate-method not being run for structs. #32
- Fix field validation errors on structs fields does not contain missing or failed configuration variable. #31

## [0.2.1]

### Changed
- Report error message from regexp.Compile if compilation fails #21

### Fixed
- Handle empty slices, strings, regular expression by nonzero,required validation tags #20, #23

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


[Unreleased]: https://github.com/elastic/go-ucfg/compare/v0.3.4...HEAD
[0.3.4]: https://github.com/elastic/go-ucfg/compare/v0.3.3...v0.3.4
[0.3.3]: https://github.com/elastic/go-ucfg/compare/v0.3.2...v0.3.3
[0.3.2]: https://github.com/elastic/go-ucfg/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/elastic/go-ucfg/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/elastic/go-ucfg/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/elastic/go-ucfg/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/elastic/go-ucfg/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/elastic/go-ucfg/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/elastic/go-ucfg/compare/v0.0.0...v0.1.0
