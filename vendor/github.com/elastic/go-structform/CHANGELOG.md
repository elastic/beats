# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

## [0.0.9]

### Added
- Added `IsZeroer` interface to allow custom types to report that they are not initialized. A structs field is not serialized if the `omitempty` struct tag is set and `IsZero()` returns true. #32

### Fixed
- Ensure `Fold` can be called when a value is given by value, but the `Folder` interface is implemented on the pointer type. #32

## [0.0.8]

### Added

- Add optional support to JSON encoder to encode a NaN or Inf floating point value to  null. (PR #28)

## [0.0.7]

### Fixed

- Fix potential use after free in string and []byte conversions. (PR #21)

## [0.0.6]

### Added
- Regenerate code with new stringer. (PR #9)
- Add support for custom unfolders when generating gotype.Unfolder. (PR #14, PR #15, PR #17)
- Add go.mod file

## [0.0.5]

### Added
- Add Reset to gotype.Unfolder. (PR #7)

## [0.0.4]

### Added
- Add SetEscapeHTML to json visitor. (PR #4)

## [0.0.3]

### Added
- Add `visitors.NilVisitor`. (Commit ab1cb2d)

### Changed
- Replace code generator with mktmlp (github.com/urso/mktmpl). (Commit 0356386)
- Introduce custom number parser. (Commit 41308dd)

### Fixed
- Fix gc failures by removing region allocator for temporary objects in decoder. Decoding into `map[string]X` with `X` being a custom go struct will require an extra alloc by now. (Commit 9b12176)
- Fix invalid cast on pointer math. (Commit ea18344)

## [0.0.2]

### Added
- Add struct tag option ",omitempty".
- Add StringConvVisitor converting all primitive values to strings.
- Move and export object visitor into visitors package

### Fixed
- Fix invalid pointer indirections in struct to array/map.

[Unreleased]: https://github.com/elastic/go-structform/compare/v0.0.9...HEAD
[0.0.9]: https://github.com/elastic/go-structform/compare/v0.0.8...v0.0.9
[0.0.8]: https://github.com/elastic/go-structform/compare/v0.0.7...v0.0.8
[0.0.7]: https://github.com/elastic/go-structform/compare/v0.0.6...v0.0.7
[0.0.6]: https://github.com/elastic/go-structform/compare/v0.0.5...v0.0.6
[0.0.5]: https://github.com/elastic/go-structform/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/elastic/go-structform/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/elastic/go-structform/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/elastic/go-structform/compare/v0.0.1...v0.0.2
