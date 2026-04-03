## 9.3.3 [beats-release-notes-9.3.3]



### Features and enhancements [beats-9.3.3-features-enhancements]


**All**

* Update OTel Collector components to v0.148.0. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49512](https://github.com/elastic/beats/issues/49512)

**Filebeat**

* Add retry back-off logic to streaming input CrowdStrike follower. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49512](https://github.com/elastic/beats/issues/49512)
* Add secret_state config to CEL input for encrypted storage of secrets accessible as state.secret. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49512](https://github.com/elastic/beats/issues/49512)

  Add a secret_state configuration field to the CEL input. When configured in a
  Fleet integration package with secret: true, the values are stored encrypted by
  Fleet. At runtime, the contents are placed at state.secret and unconditionally
  redacted in debug logs. The key &#34;secret&#34; in the plain-text state configuration
  is reserved and rejected by validation to prevent accidental unencrypted storage
  of values intended to be secret.
  
* Allow string and number arrays in httpjson chained configurations. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#16662](https://github.com/elastic/integrations/pull/16662)
* Add support for URL and URL query parsing and formatting in the Streaming input CEL environment. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49927](https://github.com/elastic/beats/pull/49927) [#49512](https://github.com/elastic/beats/issues/49512)

**Metricbeat**

* Add client secret authentication support to Azure App Insights module. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49512](https://github.com/elastic/beats/issues/49512)


### Fixes [beats-9.3.3-fixes]


**All**

* Fix grammar errors in user-facing reference docs (dashboard agreement, pipelines spelling, settings agreement, configuring usage). [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49512](https://github.com/elastic/beats/issues/49512)
* Fix duplicated word &#34;the the&#34; in ECS field descriptions (7 occurrences in fields.ecs.yml). [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49927](https://github.com/elastic/beats/pull/49927) [#49512](https://github.com/elastic/beats/issues/49512)

**Elastic agent**

* Fix an issue that could delay reporting shutdown of Agent components. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49512](https://github.com/elastic/beats/issues/49512)
* Reduce AutoOps logging from info to debug for polling. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49512](https://github.com/elastic/beats/issues/49512)

**Filebeat**

* Fix typos in crowdstrike, o365, okta, and santa module docs and field descriptions. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49512](https://github.com/elastic/beats/issues/49512)
* Fix Filestream take_over causing file re-ingestion when used with autodiscover. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49579](https://github.com/elastic/beats/issues/49579)
* Fix compatibility of the Journald input with journald/systemd versions &lt; 242. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49512](https://github.com/elastic/beats/issues/49512)
* Add rate-limit backoff to CrowdStrike streaming input oauth2 transport. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49512](https://github.com/elastic/beats/issues/49512)

  Wrap the oauth2 HTTP transport used by the CrowdStrike falcon streaming input
  with a rate-limit-aware transport that intercepts 429 responses, reads the
  Retry-After header, and backs off before retrying. This prevents the oauth2
  token refresh from generating a burst of unauthorized requests that triggers
  CrowdStrike&#39;s 15-per-minute rate limit. The discover endpoint also returns a
  retry-after hint to the session-level retry loop as a minimum wait floor.
  
* Fix duplicated word &#34;the the&#34; in mysqlenterprise module documentation. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49927](https://github.com/elastic/beats/pull/49927) [#49512](https://github.com/elastic/beats/issues/49512)
* Skip request tracer path validation when tracing is disabled to prevent input startup failures. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49927](https://github.com/elastic/beats/pull/49927) [#49512](https://github.com/elastic/beats/issues/49512)

  The startup path validation in cel, httpjson, http_endpoint, and entity
  analytics inputs checked whether the tracer config struct was non-nil rather
  than whether tracing was enabled. Integration package templates always include
  a tracer block (with enabled defaulting to false), so the struct is never nil.
  Under the agentless/otel runtime the relative tracer path resolves outside the
  permitted directory, causing all affected inputs to fail immediately even though
  tracing was disabled. The config-level Validate methods already used the correct
  enabled() guard; the startup paths now do the same.
  
* Fix Filebeat crash loop when running under Elastic Agent and taking too long to initialise. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49927](https://github.com/elastic/beats/pull/49927) [#49512](https://github.com/elastic/beats/issues/49512)

**Heartbeat**

* Fix &#34;realtive&#34; → &#34;relative&#34; typo in heartbeat browser field description. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49927](https://github.com/elastic/beats/pull/49927) [#49512](https://github.com/elastic/beats/issues/49512)

**Libbeat**

* Fixed a bug where escaped characters in syslog structured data caused an EOF error. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49927](https://github.com/elastic/beats/pull/49927) [#43944](https://github.com/elastic/beats/issues/43944)

**Metricbeat**

* Fix unnecessary Windows filesystem metricset errors from non-existent volumes. [#49553](https://github.com/elastic/beats/pull/49553) 

  Fixes an issue where filesystem metric collection on Windows could report errors for volumes that are no longer present. Updated to gosigar v0.14.4.

**Packetbeat**

* Fix possessive &#34;its&#34; and &#34;currnet&#34; typos in packetbeat configuration and field descriptions. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49512](https://github.com/elastic/beats/issues/49512)

**Winlogbeat**

* Skip record ID gap detection for forwarded Windows events. [#49796](https://github.com/elastic/beats/pull/49796) [#49904](https://github.com/elastic/beats/pull/49904) [#49905](https://github.com/elastic/beats/pull/49905) [#49906](https://github.com/elastic/beats/pull/49906) [#49927](https://github.com/elastic/beats/pull/49927) [#49512](https://github.com/elastic/beats/issues/49512)

