# Change Log
All notable changes to this project will be documented in this file based on the
[Keep a Changelog](http://keepachangelog.com/) Standard.

## [1.0.1](https://github.com/elastic/beats/compare/1.0.0...1.0.1)

### Backward Compatibility Breaks
- Removal of partial_line_waiting config as it was not working as expected. #296

### Bugfixes
- Fix force_close_files in case renamed file appeared very fast elastic/filebeat#302

### Added
- Validate harvester input_type and make selection fully dependent on input_type definition.

### Deprecated

## [1.0.0](https://github.com/elastic/filebeat/compare/1.0.0-rc2...1.0.0) - 2015-11-24

### Backward Compatibility Breaks

### Bugfixes
- Fix problem that harvesters stopped reading after some time and filebeat stopped processing events #257
- Fix line truncating by internal buffers being reused by accident #258
- Set default ignore_older to 24 hours #282

### Added

### Deprecated

## [1.0.0-rc2](https://github.com/elastic/filebeat/compare/1.0.0-rc1...1.0.0-rc2) - 2015-11-17

### Backward Compatibility Breaks
- Removed utf-16be-bom encoding support. Support will be added with fix for #205
- Rename force_close_windows_files to force_close_files and make it available for all platforms.

### Bugfixes
- Filebeat will now exit if a configuration error is detected. #198
- Fix to enable prospector to harvest existing files that are modified. #199
- Improve line reading and encoding to better keep track of file offsets based
  on encoding. #224
- Set input_type by default to "log"

### Added
- Handling end of line under windows was improved #233

### Deprecated

## [1.0.0-rc1](https://github.com/elastic/filebeat/compare/1.0.0-beta4...1.0.0-rc1)

### Backward Compatibility Breaks
- Rename the timestamp field with @timestamp #168
- Rename tail_on_rotate prospector config to tail_files
- Removal of line field in event. Line number was not correct and does not add value. #217

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
