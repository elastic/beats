# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

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


[Unreleased]: https://github.com/urso/ucfg/compare/v0.1.0...HEAD
[0.1.1]: https://github.com/urso/ucfg/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/urso/ucfg/compare/v0.0.0...v0.1.0
