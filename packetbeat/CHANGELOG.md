# Change Log
All notable changes to this project will be documented in this file based on the
[Keep a Changelog](http://keepachangelog.com/) Standard.

## [1.0.1](https://github.com/elastic/beats/compare/1.0.0...1.0.1)

### Backward Compatibility Breaks

### Bugfixes
- Fix panic on nil in redis protocol parser. #384
- Fix errors redis parser when messages are split in multiple TCP segments. #402
- Fix errors in redis parser when length prefixed strings contain sequences of CRLF. #402
- Fix errors in redis parser when dealing with nested arrays. #402
- Improve MongoDB message correlation. #377
- Fix TCP connection state being reused after dropping due to gap in stream. #342

### Added
- Added redis pipelining support. #402
- Add http pipelining support. #453

### Improvements
- Fix panic on nil in redis protocol parser. #384
- Fix errors redis parser when messages are split in multiple TCP segments. #402
- Fix errors in redis parser when length prefixed strings contain sequences of CRLF. #402
- Fix errors in redis parser when dealing with nested arrays. #402

### Added
- Added redis pipelining support. #402
- Improve redis parser performance. #442

### Deprecated

## [1.0.0](https://github.com/elastic/packetbeat/compare/1.0.0-rc2...1.0.0) 2015-11-24

### Backward Compatibility Breaks

### Bugfixes

### Added

### Improvements

### Deprecated

## [1.0.0-rc2](https://github.com/elastic/packetbeat/compare/1.0.0-rc1...1.0.0-rc2) 2015-11-17

### Backward Compatibility Breaks

### Bugfixes
- Packetbeat will now exit if a configuration error is detected. #357
- Fixed an issue handling DNS requests containing no questions. #369

### Added

### Deprecated

## [1.0.0-rc1](https://github.com/elastic/packetbeat/compare/1.0.0-beta4...1.0.0-rc1) 2015-11-04

### Backward Compatibility Breaks
- Rename timestamp field with @timestamp. #343

### Bugfixes
- Close file descriptors used to monitor processes. #337
- Remove old RPM spec file. It moved to elastic/beats-packer. #334

### Added

### Deprecated

## [1.0.0-beta4](https://github.com/elastic/packetbeat/compare/1.0.0-beta3...1.0.0-beta4) 2015-10-21

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
