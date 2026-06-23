## 9.4.3 [beats-release-notes-9.4.3]

_This release also includes: [Deprecations](/release-notes/deprecations.md#beats-9.4.3-deprecations)._


### Features and enhancements [beats-9.4.3-features-enhancements]


**All**

* Migrate from deprecated github.com/docker/docker to github.com/moby/moby split modules. [#50300](https://github.com/elastic/beats/pull/50300) 

**Beats**

* Allow TLS certificate and CA hot-reload without restarting. [#50444](https://github.com/elastic/beats/pull/50444) 

**Metricbeat**

* Add `data_stream` field alongside `aliases` in cat_shards resolved_indices for metricbeat autoops_es module.  


### Fixes [beats-9.4.3-fixes]


**All**

* Upgrade to Go 1.26.4.  

**Beats**

* Disable TLS certificate hot-reload by default on patch branches. [#51104](https://github.com/elastic/beats/pull/51104) 
* Fix panic in the Elastic Agent V2 manager when reloading with no output unit.  

**Elastic agent**

* Fix beat receivers adding .0 to whole floating point numbers when encoding to json.  [#14610](https://github.com/elastic/elastic-agent/issues/14610)

**Filebeat**

* Fix request tracer path validation for cel, httpjson, http_endpoint, and entityanalytics inputs when filebeat runs as an OTel receiver. [#50581](https://github.com/elastic/beats/pull/50581) 
* Fix goroutine leak in filestream task group. [#50839](https://github.com/elastic/beats/pull/50839) [#50824](https://github.com/elastic/beats/issues/50824)
* Fix Okta entity analytics OAuth2 jwk_json token refresh failure.  [#50949](https://github.com/elastic/beats/issues/50949)

  The legacy Okta entity analytics provider did not store the JWK bytes when
  oauth2.jwk_json was configured, causing token refresh to fail with
  &#34;error decoding JWK: unexpected end of JSON input&#34;. Also add a cached-token
  validity check to avoid regenerating the JWT on every API request.
  
* Cache Okta OAuth2 token in cel and httpjson to avoid unnecessary JWT regeneration.  

  The oktaTokenSource.Token() method in the cel and httpjson inputs
  unconditionally regenerated a JWT and exchanged it for a bearer token
  on every call, even when the cached token was still valid. Add a
  token validity check to skip regeneration when unnecessary.
  
* Fix WebSocket reconnect loop ignoring context cancellation with infinite retries.  
* Fix filestream registry leak on file renames.  
* Fix WebSocket input hanging on shutdown when server stalls and keep_alive is disabled.  
* Fix handling of user-agent header when using OAuth2.0 authentication.  
* Guard event.original rename in azure module ingest pipelines to prevent &#34;field already exists&#34; error when the field is pre-populated. [#51271](https://github.com/elastic/beats/pull/51271) 

**Libbeat**

* Fix bulk indexing failures that could occur when using the Elasticsearch output with some non-Elastic backends. [#49557](https://github.com/elastic/beats/pull/49557) 
* Fix data race in add_cloud_metadata processor when fetching metadata from multiple providers concurrently.  

**Metricbeat**

* Clamp autoops_es *_latency_in_millis metrics to the sampling interval so a single-sample latency can never exceed the wall-clock time between samples (fixes #2471). [#50688](https://github.com/elastic/beats/pull/50688) 

**Osquerybeat**

* Respect osquery pack query platform filters for Live Query actions. [#50585](https://github.com/elastic/beats/pull/50585) 

**Winlogbeat**

* Treat RPC_S_UNKNOWN_IF (1717) as a recoverable error so Winlogbeat reopens the event log session instead of exiting on this transient RPC error.  
* Fix Winlogbeat record ID gap retries reopening from stale checkpoints.  
* Fix winlog XML rendering on Windows arm64.  

