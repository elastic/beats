# Change Log
All notable changes to this project will be documented in this file based on the
[Keep a Changelog](http://keepachangelog.com/) Standard.

## [1.0.1](https://github.com/elastic/topbeat/compare/1.0.0...1.0.1)

### Backward Compatibility Breaks

### Bugfixes

### Added

### Deprecated

## [1.0.0](https://github.com/elastic/topbeat/compare/1.0.0-rc2...1.0.0) - 2015-11-24

### Backward Compatibility Breaks
- Change proc type to process #138

### Bugfixes

### Added

### Deprecated

## [1.0.0-rc2](https://github.com/elastic/topbeat/compare/1.0.0-rc1...1.0.0-rc2) - 2015-11-17

### Backward Compatibility Breaks

### Bugfixes
- Fix leak of Windows handles. #98
- Fix memory leak of process information. #104

### Added
- Export mem.actual_used_p and swap.actual_used_p #93
- Configure what type of statistics to be exported #99

### Deprecated

## [1.0.0-rc1](https://github.com/elastic/topbeat/compare/1.0.0-beta4...1.0.0-rc1) - 2015-11-04

### Backward Compatibility Breaks
- Rename timestamp field with @timestamp for a better integration with
Logstash. #80

### Bugfixes
- Don't wait for one period until shutdown #75

### Added

### Deprecated

## [1.0.0-beta4](https://github.com/elastic/topbeat/compare/1.0.0-beta3...1.0.0-beta4) - 2015-10-22

### Backward Compatibility Breaks
- Percentage fields (e.g user_p) are exported as a float between 0 and 1 #34

### Bugfixes
- Don't divide the reported memory by an extra 1024 #60

### Added
- Document fields in a standardized format (etc/fields.yml) #34
- Updated to use new libbeat Publisher #37 #41
- Update to go 1.5.1 #43
- Updated configuration files with comments for all options #65
- Documentation improvements
