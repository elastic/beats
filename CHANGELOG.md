# Change Log
All notable changes to this project will be documented in this file based on the
[Keep a Changelog](http://keepachangelog.com/) Standard.


## [Unreleased](https://github.com/elastic/libbeat/compare/1.0.0-beta3...HEAD)

### Backward Compatibility Breaks

### Bugfixes
- Determine Elasticsearch index for an event based on UTC time [#81](https://github.com/elastic/libbeat/issues/81)

### Added
- Add logstash output plugin

### Improvements
- Add tls configuration support to elasticsearch and logstash outputers

### Deprecated

 * host and port configuration options are deprecated. They are replaced by the hosts
 configuration option.
