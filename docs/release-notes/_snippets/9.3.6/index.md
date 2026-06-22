## 9.3.6 [beats-release-notes-9.3.6]

_This release also includes: [Deprecations](/release-notes/deprecations.md#beats-9.3.6-deprecations)._


### Features and enhancements [beats-9.3.6-features-enhancements]


**All**

* Migrate from deprecated github.com/docker/docker to github.com/moby/moby split modules. [#50300](https://github.com/elastic/beats/pull/50300) 

**Beats**

* Allow TLS certificate and CA hot-reload without restarting. [#51302](https://github.com/elastic/beats/pull/51302) [#49643](https://github.com/elastic/beats/pull/49643) [#51369](https://github.com/elastic/beats/pull/51369) [#51370](https://github.com/elastic/beats/pull/51370) [#51373](https://github.com/elastic/beats/pull/51373) [#51374](https://github.com/elastic/beats/pull/51374) [#51255](https://github.com/elastic/beats/pull/51255) [#34074](https://github.com/elastic/beats/issues/34074)

**Metricbeat**

* Add `data_stream` field alongside `aliases` in cat_shards resolved_indices for metricbeat autoops_es module. [#51302](https://github.com/elastic/beats/pull/51302) [#49643](https://github.com/elastic/beats/pull/49643) [#51369](https://github.com/elastic/beats/pull/51369) [#51370](https://github.com/elastic/beats/pull/51370) [#51373](https://github.com/elastic/beats/pull/51373) [#51374](https://github.com/elastic/beats/pull/51374) [#51255](https://github.com/elastic/beats/pull/51255) [#34074](https://github.com/elastic/beats/issues/34074)


### Fixes [beats-9.3.6-fixes]


**All**

* Upgrade to Go 1.26.4. [#51302](https://github.com/elastic/beats/pull/51302) [#49643](https://github.com/elastic/beats/pull/49643) [#51369](https://github.com/elastic/beats/pull/51369) [#51370](https://github.com/elastic/beats/pull/51370) [#51373](https://github.com/elastic/beats/pull/51373) [#51374](https://github.com/elastic/beats/pull/51374) [#51255](https://github.com/elastic/beats/pull/51255) [#34074](https://github.com/elastic/beats/issues/34074)

**Beats**

* Disable TLS certificate hot-reload by default on patch branches. [#51103](https://github.com/elastic/beats/pull/51103) 
* Fix panic in the Elastic Agent V2 manager when reloading with no output unit. [#51302](https://github.com/elastic/beats/pull/51302) [#49643](https://github.com/elastic/beats/pull/49643) [#51369](https://github.com/elastic/beats/pull/51369) [#51370](https://github.com/elastic/beats/pull/51370) [#51373](https://github.com/elastic/beats/pull/51373) [#51374](https://github.com/elastic/beats/pull/51374) [#51255](https://github.com/elastic/beats/pull/51255) [#34074](https://github.com/elastic/beats/issues/34074)

**Elastic agent**

* Fix beat receivers adding .0 to whole floating point numbers when encoding to json. [#51302](https://github.com/elastic/beats/pull/51302) [#49643](https://github.com/elastic/beats/pull/49643) [#51369](https://github.com/elastic/beats/pull/51369) [#51370](https://github.com/elastic/beats/pull/51370) [#51373](https://github.com/elastic/beats/pull/51373) [#51374](https://github.com/elastic/beats/pull/51374) [#51255](https://github.com/elastic/beats/pull/51255) [#14610](https://github.com/elastic/elastic-agent/issues/14610)

**Filebeat**

* Fix request tracer path validation for cel, httpjson, http_endpoint, and entityanalytics inputs when filebeat runs as an OTel receiver. [#51302](https://github.com/elastic/beats/pull/51302) [#49643](https://github.com/elastic/beats/pull/49643) [#51369](https://github.com/elastic/beats/pull/51369) [#51370](https://github.com/elastic/beats/pull/51370) [#51373](https://github.com/elastic/beats/pull/51373) [#51374](https://github.com/elastic/beats/pull/51374) [#51255](https://github.com/elastic/beats/pull/51255) [#34074](https://github.com/elastic/beats/issues/34074)
* Fix goroutine leak in filestream task group. [#50839](https://github.com/elastic/beats/pull/50839) [#50824](https://github.com/elastic/beats/issues/50824)
* Fix Okta entity analytics OAuth2 jwk_json token refresh failure. [#51302](https://github.com/elastic/beats/pull/51302) [#49643](https://github.com/elastic/beats/pull/49643) [#51369](https://github.com/elastic/beats/pull/51369) [#51370](https://github.com/elastic/beats/pull/51370) [#51373](https://github.com/elastic/beats/pull/51373) [#51374](https://github.com/elastic/beats/pull/51374) [#51255](https://github.com/elastic/beats/pull/51255) [#50949](https://github.com/elastic/beats/issues/50949)

  The legacy Okta entity analytics provider did not store the JWK bytes when
  oauth2.jwk_json was configured, causing token refresh to fail with
  &#34;error decoding JWK: unexpected end of JSON input&#34;. Also add a cached-token
  validity check to avoid regenerating the JWT on every API request.
  
* Cache Okta OAuth2 token in cel and httpjson to avoid unnecessary JWT regeneration. [#51302](https://github.com/elastic/beats/pull/51302) [#49643](https://github.com/elastic/beats/pull/49643) [#51369](https://github.com/elastic/beats/pull/51369) [#51370](https://github.com/elastic/beats/pull/51370) [#51373](https://github.com/elastic/beats/pull/51373) [#51374](https://github.com/elastic/beats/pull/51374) [#51255](https://github.com/elastic/beats/pull/51255) [#34074](https://github.com/elastic/beats/issues/34074)

  The oktaTokenSource.Token() method in the cel and httpjson inputs
  unconditionally regenerated a JWT and exchanged it for a bearer token
  on every call, even when the cached token was still valid. Add a
  token validity check to skip regeneration when unnecessary.
  
* Fix filestream registry leak on file renames. [#51302](https://github.com/elastic/beats/pull/51302) [#49643](https://github.com/elastic/beats/pull/49643) [#51369](https://github.com/elastic/beats/pull/51369) [#51370](https://github.com/elastic/beats/pull/51370) [#51373](https://github.com/elastic/beats/pull/51373) [#51374](https://github.com/elastic/beats/pull/51374) [#51255](https://github.com/elastic/beats/pull/51255) [#34074](https://github.com/elastic/beats/issues/34074)
* Guard event.original rename in azure module ingest pipelines to prevent &#34;field already exists&#34; error when the field is pre-populated. [#51271](https://github.com/elastic/beats/pull/51271) 

**Libbeat**

* Fix data race in add_cloud_metadata processor when fetching metadata from multiple providers concurrently. [#51302](https://github.com/elastic/beats/pull/51302) [#49643](https://github.com/elastic/beats/pull/49643) [#51369](https://github.com/elastic/beats/pull/51369) [#51370](https://github.com/elastic/beats/pull/51370) [#51373](https://github.com/elastic/beats/pull/51373) [#51374](https://github.com/elastic/beats/pull/51374) [#51255](https://github.com/elastic/beats/pull/51255) [#34074](https://github.com/elastic/beats/issues/34074)

**Metricbeat**

* Clamp autoops_es *_latency_in_millis metrics to the sampling interval so a single-sample latency can never exceed the wall-clock time between samples (fixes #2471). [#51302](https://github.com/elastic/beats/pull/51302) [#49643](https://github.com/elastic/beats/pull/49643) [#51369](https://github.com/elastic/beats/pull/51369) [#51370](https://github.com/elastic/beats/pull/51370) [#51373](https://github.com/elastic/beats/pull/51373) [#51374](https://github.com/elastic/beats/pull/51374) [#51255](https://github.com/elastic/beats/pull/51255) [#34074](https://github.com/elastic/beats/issues/34074)

**Osquerybeat**

* Respect osquery pack query platform filters for Live Query actions. [#51302](https://github.com/elastic/beats/pull/51302) [#49643](https://github.com/elastic/beats/pull/49643) [#51369](https://github.com/elastic/beats/pull/51369) [#51370](https://github.com/elastic/beats/pull/51370) [#51373](https://github.com/elastic/beats/pull/51373) [#51374](https://github.com/elastic/beats/pull/51374) [#51255](https://github.com/elastic/beats/pull/51255) [#34074](https://github.com/elastic/beats/issues/34074)

**Packetbeat**

* Fix janitor goroutine leaks and decoder cleanup lifecycle on route changes. [#48836](https://github.com/elastic/beats/pull/48836) 

**Winlogbeat**

* Treat RPC_S_UNKNOWN_IF (1717) as a recoverable error so Winlogbeat reopens the event log session instead of exiting on this transient RPC error. [#51302](https://github.com/elastic/beats/pull/51302) [#49643](https://github.com/elastic/beats/pull/49643) [#51369](https://github.com/elastic/beats/pull/51369) [#51370](https://github.com/elastic/beats/pull/51370) [#51373](https://github.com/elastic/beats/pull/51373) [#51374](https://github.com/elastic/beats/pull/51374) [#51255](https://github.com/elastic/beats/pull/51255) [#34074](https://github.com/elastic/beats/issues/34074)
* Fix Winlogbeat record ID gap retries reopening from stale checkpoints. [#51302](https://github.com/elastic/beats/pull/51302) [#49643](https://github.com/elastic/beats/pull/49643) [#51369](https://github.com/elastic/beats/pull/51369) [#51370](https://github.com/elastic/beats/pull/51370) [#51373](https://github.com/elastic/beats/pull/51373) [#51374](https://github.com/elastic/beats/pull/51374) [#51255](https://github.com/elastic/beats/pull/51255) [#34074](https://github.com/elastic/beats/issues/34074)
* Fix winlog XML rendering on Windows arm64. [#51302](https://github.com/elastic/beats/pull/51302) [#49643](https://github.com/elastic/beats/pull/49643) [#51369](https://github.com/elastic/beats/pull/51369) [#51370](https://github.com/elastic/beats/pull/51370) [#51373](https://github.com/elastic/beats/pull/51373) [#51374](https://github.com/elastic/beats/pull/51374) [#51255](https://github.com/elastic/beats/pull/51255) [#34074](https://github.com/elastic/beats/issues/34074)

