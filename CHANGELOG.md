# Change Log
All notable changes to this project will be documented in this file based on the
[Keep a Changelog](http://keepachangelog.com/) Standard.

## [Unreleased](https://github.com/elastic/topbeat/compare/1.0.0-beta4...HEAD)

### Backward Compatibility Breaks

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
