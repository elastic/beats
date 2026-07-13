## 9.3.6 [beats-release-notes-9.3.6]

_This release also includes: [Deprecations](/release-notes/deprecations.md#beats-9.3.6-deprecations)._


### Features and enhancements [beats-9.3.6-features-enhancements]


**All**

* Migrate from the deprecated `github.com/docker/docker` module to the `github.com/moby/moby` split modules. [#50300](https://github.com/elastic/beats/pull/50300) 

**Beats**

* Allow TLS certificate and CA hot-reload without restarting. [#50444](https://github.com/elastic/beats/pull/50444) [#34074](https://github.com/elastic/beats/issues/34074)

**Metricbeat**

* Add `data_stream` field alongside `aliases` in `cat_shards` for the Metricbeat `autoops_es` module. [#51108](https://github.com/elastic/beats/pull/51108) 


### Fixes [beats-9.3.6-fixes]


**All**

* Upgrade Go to v1.26.4. [#51115](https://github.com/elastic/beats/pull/51115) 

**Beats**

* Disable TLS certificate hot-reload by default on patch branches. [#51103](https://github.com/elastic/beats/pull/51103) 
* Fix panic in the Elastic Agent V2 manager when reloading with no output unit. [#51247](https://github.com/elastic/beats/pull/51247) 

**Elastic agent**

* Fix Beat receivers adding `.0` to whole floating point numbers when encoding to JSON. [#51018](https://github.com/elastic/beats/pull/51018) [#14610](https://github.com/elastic/elastic-agent/issues/14610)

**Filebeat**

* Fix request tracer path validation for the `cel`, `httpjson`, `http_endpoint`, and `entityanalytics` inputs when Filebeat runs as an OTel receiver. [#50581](https://github.com/elastic/beats/pull/50581) 
* Fix goroutine leak in filestream task group. [#50839](https://github.com/elastic/beats/pull/50839) [#50824](https://github.com/elastic/beats/issues/50824)
* Fix Okta entity analytics OAuth2 `jwk_json` token refresh failure. [#51079](https://github.com/elastic/beats/pull/51079) [#50949](https://github.com/elastic/beats/issues/50949)
* Cache Okta OAuth2 token in `cel` and `httpjson` to avoid unnecessary JWT regeneration. [#51080](https://github.com/elastic/beats/pull/51080) 
* Fix filestream registry leak on file renames. [#51254](https://github.com/elastic/beats/pull/51254) 
* Guard `event.original` rename in Azure module ingest pipelines to prevent a "field already exists" error when the field is pre-populated. [#51271](https://github.com/elastic/beats/pull/51271) 

**Libbeat**

* Fix a data race in the `add_cloud_metadata` processor when fetching metadata from multiple providers concurrently. [#51295](https://github.com/elastic/beats/pull/51295) 

**Metricbeat**

* Clamp `autoops_es` `*_latency_in_millis` metrics to the sampling interval so a single-sample latency can never exceed the wall-clock time between samples. [#50688](https://github.com/elastic/beats/pull/50688) 

**Osquerybeat**

* Respect Osquery pack query platform filters for Live Query actions. [#50585](https://github.com/elastic/beats/pull/50585) 

**Packetbeat**

* Fix janitor goroutine leaks and decoder cleanup lifecycle on route changes. [#48836](https://github.com/elastic/beats/pull/48836) 

**Winlogbeat**

* Treat RPC_S_UNKNOWN_IF (1717) as a recoverable error so Winlogbeat reopens the event log session instead of exiting on this transient RPC error. [#50875](https://github.com/elastic/beats/pull/50875) 
* Fix Winlogbeat record ID gap retries reopening from stale checkpoints. [#50917](https://github.com/elastic/beats/pull/50917) 
* Fix winlog XML rendering on Windows arm64. [#51055](https://github.com/elastic/beats/pull/51055) 
