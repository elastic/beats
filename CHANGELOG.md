# Change Log
All notable changes to this project will be documented in this file based on the
[Keep a Changelog](http://keepachangelog.com/) Standard.

## [Unreleased](https://github.com/elastic/libbeat/compare/1.0.0-rc1...HEAD)

### Backward Compatibility Breaks

### Bugfixes
- Packetbeat will now exit if a configuration error is detected. #357

### Added

### Deprecated

## [1.0.0-rc1](https://github.com/elastic/libbeat/compare/1.0.0-beta4...1.0.0-rc1) 2015-11-04

### Backward Compatibility Breaks
- Rename timestamp field with @timestamp. #343

### Bugfixes
- Close file descriptors used to monitor processes. #337
- Remove old RPM spec file. It moved to elastic/beats-packer. #334

### Added

### Deprecated

## [1.0.0-beta4](https://github.com/elastic/libbeat/compare/1.0.0-beta3...1.0.0-beta4) 2015-10-21

### Backward Compatibility Breaks
- Renamed http module config file option 'strip_authorization' to 'redact_authorization'
- Save_topology is set to false by default
- Rename elasticsearch index to [packetbeat-]YYYY.MM.DD

### Bugfixes
- Support for lower-case header names when redacting http authorization headers
- Redact proxy-authorization if redact-authorization is set
- Fix some multithreading issues #203
- Fix negative response time #216
- Fix memcache TCP connection being nil after dropping stream data. #299
- Add missing DNS protocol configuration to documentation #269

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
- Made the `-devices` option more user friendly in case `sudo` is not used.
  Issue #296.
- Publish expired DNS transactions #301
- Update protocol guide to libbeat changes
- Add protocol registration to new protocol guide
- Make transaction timeouts configurable #300
- Add direction field to the exported fields #317

### Deprecated
