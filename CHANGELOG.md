# Change Log
All notable changes to this project will be documented in this file based on the
[Keep a Changelog](http://keepachangelog.com/) Standard.

## [Unreleased](https://github.com/elastic/packetbeat/compare/1.0.0-beta3...HEAD)

### Backward Compatibility Breaks
- renamed http module config file option 'strip_authorization' to 'redact_authorization'
- save_topology is set to false by default
- rename elasticsearch index to [packetbeat-]YYYY.MM.DD

### Bugfixes
- Support for lower-case header names when redacting http authorization headers
- Redact proxy-authorization if redact-authorization is set
- Fix some multithreading issues #203
- Fix negative response time #216

### Added
- add [.editorconfig file](http://editorconfig.org/)
- add (experimental/unsupported?) saltstack files
- Sample config file cleanup
- Moved common documentation to [libbeat repository](https://github.com/elastic/libbeat)
- Update build to go 1.5.1
- Adding device descriptions to the -device output.
- Generate coverage for system tests
- Move go-daemon dependency to beats-packer
- Rename integration tests to system tests
- Publish expired DNS transactions #301

### Deprecated
