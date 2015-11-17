# Change Log
All notable changes to this project will be documented in this file based on the
[Keep a Changelog](http://keepachangelog.com/) Standard.

## [Unreleased](https://github.com/elastic/libbeat/compare/1.0.0-rc2...HEAD)

### Backward Compatibility Breaks

### Bugfixes

### Added

### Deprecated

## [1.0.0-rc2](https://github.com/elastic/libbeat/compare/1.0.0-rc1...1.0.0-rc2)

### Backward Compatibility Breaks

- The `shipper` output field is renamed to `beat.name`. #285
- Use of `enabled` as a configuration option for outputs (elasticsearch,
  logstash, etc.) has been removed. #264
- Use of `disabled` as a configuration option for tls has been removed. #264
- The `-test` command line flag was renamed to `-configtest`. #264
- Disable geoip by default. To enable it uncomment in config file. #305

### Bugfixes
- Disable logging to stderr after configuration phase. #276
- Set the default file logging path when not set in config. #275
- Fix bug silently dropping records based on current window size. elastic/filebeat#226
- Fix direction field in published events. #300
- Fix elasticsearch structured errors breaking error handling. #309

### Added

- Added `beat.hostname` to contain the hostname where the Beat is running on as
  returned by the operating system. #285
- Added timestamp for file logging. #291

### Deprecated

## [1.0.0-rc1](https://github.com/elastic/libbeat/compare/1.0.0-beta4...1.0.0-rc1)

### Backward Compatibility Breaks
- Rename timestamp field with @timestamp. #237


### Bugfixes
- Use stderr for console log output. #219
- Handle empty event array in publisher. #207
- Respect '*' debug selector in IsDebug. #226 (elastic/packetbeat#339)
- Limit number of workers for Elasticsearch output. elastic/packetbeat#226
- On Windows, remove service related error message when running in the console. #242
- Fix waitRetry no configured in single output mode configuration. elastic/filebeat#144
- Use http as the default scheme in the elasticsearch hosts #253
- Respect max bulk size if bulk publisher (collector) is disabled or sync flag is set.
- Always evaluate status code from Elasticsearch responses when indexing events. #192
- Use bulk_max_size configuration option instead of bulk_size. #256
- Fix max_retries=0 (no retries) configuration option. #266
- Filename used for file based logging now defaults to beat name. #267

### Added
- Add Console output plugin. #218
- Add timestamp to log messages #245
- Send @metadata.beat to Logstash instead of @metadata.index to prevent
  possible name clashes and give user full control over index name used for
  Elasticsearch
- Add logging messages for bulk publishing in case of error #229
- Add option to configure number of parallel workers publishing to Elasticsearch
  or Logstash.
- Set default bulk size for Elasticsearch output to 50.
- Set default http timeout for Elasticsearch to 90s.
- Improve publish retry if sync flag is set by retrying only up to max bulk size
  events instead of all events to be published.

### Deprecated


## [1.0.0-beta4](https://github.com/elastic/libbeat/compare/1.0.0-beta3...1.0.0-beta4)

### Backward Compatibility Breaks
- Update tls config options naming from dash to underline #162
- Feature/output modes: Introduction of PublishEvent(s) to be used by beats #118 #115

### Bugfixes
- Determine Elasticsearch index for an event based on UTC time #81
- Fixing ES output's defaultDeadTimeout so that it is 60 seconds #103
- ES outputer: fix timestamp conversion #91
- Fix TLS insecure config option #239
- ES outputer: check bulk API per item status code for retransmit on failure.

### Added
- Add logstash output plugin #151
- Integration tests for Beat -> Logstash -> Elasticsearch added #195 #188 #168 #137 #128 #112
- Large updates and improvements to the documentation
- Add direction field to publisher output to indicate inbound/outbound transactions #150

### Improvements
- Add tls configuration support to elasticsearch and logstash outputers #139
- All external dependencies were updated to the latest version. Update to Golang 1.5.1 #162
- Guarantee ES index is based in UTC time zone #164
- Cache: optional per element timeout #144
- Make it possible to set hosts in different ways. #135
- Expose more TLS config options #124
- Use the Beat name in the default configuration file path #99

### Deprecated
- Redis output was deprecated #169 #145
- Host and port configuration options are deprecated. They are replaced by the hosts
 configuration option. #141
