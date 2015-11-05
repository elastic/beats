# Change Log
All notable changes to this project will be documented in this file based on the
[Keep a Changelog](http://keepachangelog.com/) Standard.


## [Unreleased](https://github.com/elastic/libbeat/compare/1.0.0-rc1...HEAD)

### Backward Compatibility Breaks

### Bugfixes
- Filebeat will now exit if a configuration error is detected. #198
- Fix to enable prospector to harvest existing files that are modified. #199

### Added

### Deprecated

## [1.0.0-rc1](https://github.com/elastic/libbeat/compare/1.0.0-beta4...1.0.0-rc1)

### Backward Compatibility Breaks
- Rename the timestamp field with @timestamp #168
- Rename tail_on_rotate prospector config to tail_files

### Bugfixes
- Omit 'fields' from event JSON when null. #126
- Make offset and line value of type long in elasticsearch template to prevent overflow. #140
- Fix locking files for writing behaviour. #156
- Introduce 'document_type' config option per prospector to define document type
  for event stored in elasticsearch. #133
- Add 'input_type' field to published events reporting the prospector type being used. #133
- Fix high CPU usage when not connected to Elasticsearch or Logstash. #144
- Fix issue that files were not crawled anymore when encoding was set to something other then plain. #182

### Added
- Introduction of backoff, backoff_factor, max_backoff, partial_line_waiting, force_close_windows_files
  config variables to make crawling more configurable.

### Improvements
- All Godeps dependencies were updated to master on 2015-10-21 [#122]
- Set default value for ignore_older config to 10 minutes. #164
- Added the fields_under_root setting to optionally store the custom fields top
level in the output dictionary. #188
- Add more encodings by using x/text/encodings/htmlindex package to select
  encoding by name.

### Deprecated


## [1.0.0-beta4](https://github.com/elastic/topbeat/compare/13678f4...1.0.0-beta4) - 2015-10-22
This was the first release of Filebeat.
