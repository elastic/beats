# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

## [0.5.1]

### Fixed
- Fix: an issue with the Cyclic reference algorithm when a direct reference was pointing
  to another reference. #100

## [0.5.0]

### Added
- Detect cyclic reference and allow to search top level key with the other resolvers. #97
- Allow to diff keys of two different configuration #93

## [0.4.6]

### Added
- Introduce ,ignore struct tag option to optionally ignore exported fields. #89
- Add support for custom Unpacker method with `*Config` being convertible to first parameter. The custom method must be compatible to `ConfigUnpacker`. #90

### Fixed
- Ignore private struct fields when merging a struct into a config. #89

## [0.4.5]

### Changed
- merging sub-configs enforces strict variable expansion #85

### Fixed
- fix merging nil sub-configs #85

## [0.4.4]

### Added
- Add support for pure array config files #82

### Changed
- Invalid top-level types return non-critical error (no stack-trace) on merge #82

### Fixed
- Fix panic when merging or creating a config from nil interface value #82

## [0.4.3]

### Changed
- Add per element type stop set for handling unquoted strings (reduces need for quoting strings in environment variables) #80

### Fixed
- fix issue unpacking array from environment variable into struct array fields #80
- fix unparsed values being used for unpacking #80

## [0.4.2]

### Fixed
- Treat `,` character as only special character in non quoted top-level strings. #78

## [0.4.1]

### Fixed
- Fix parsing empty string or nil objects from environment variables. #76

## [0.4.0]

### Added
- Syntax for passing lists and dictionaries to flags. #72
- Add Unpacker interface specializations for primitive types. #73
- Variable expansion parsing lists and dictionaries with parser introduced in
  #72. #74

### Fixed
- Fix Unpacker interface not applied if some 'old' value is already present on
  target and is struct implementing Unpack. #73

## [0.3.7]

### Fixed
- Fix int/uint to float type conversation. #68
- Fix primitive type unpacking for variables expanded from environment variables
  or strings read/created by config file parsers. #67

## [0.3.6]

### Fixed
- Fix duplicate key error when normalizing tables. #63

## [0.3.5]

### Fixed
- Fix merging array values. #59
- Fix initializing empty array values. #58

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


[Unreleased]: https://github.com/elastic/go-ucfg/compare/v0.5.1...HEAD
[0.5.1]: https://github.com/elastic/go-ucfg/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/elastic/go-ucfg/compare/v0.4.6...v0.5.0
[0.4.6]: https://github.com/elastic/go-ucfg/compare/v0.4.5...v0.4.6
[0.4.5]: https://github.com/elastic/go-ucfg/compare/v0.4.4...v0.4.5
[0.4.4]: https://github.com/elastic/go-ucfg/compare/v0.4.3...v0.4.4
[0.4.3]: https://github.com/elastic/go-ucfg/compare/v0.4.2...v0.4.3
[0.4.2]: https://github.com/elastic/go-ucfg/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/elastic/go-ucfg/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/elastic/go-ucfg/compare/v0.3.7...v0.4.0
[0.3.7]: https://github.com/elastic/go-ucfg/compare/v0.3.6...v0.3.7
[0.3.6]: https://github.com/elastic/go-ucfg/compare/v0.3.5...v0.3.6
[0.3.5]: https://github.com/elastic/go-ucfg/compare/v0.3.4...v0.3.5
[0.3.4]: https://github.com/elastic/go-ucfg/compare/v0.3.3...v0.3.4
[0.3.3]: https://github.com/elastic/go-ucfg/compare/v0.3.2...v0.3.3
[0.3.2]: https://github.com/elastic/go-ucfg/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/elastic/go-ucfg/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/elastic/go-ucfg/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/elastic/go-ucfg/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/elastic/go-ucfg/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/elastic/go-ucfg/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/elastic/go-ucfg/compare/v0.0.0...v0.1.0
