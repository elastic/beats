## 9.2.8 [beats-release-notes-9.2.8]



### Features and enhancements [beats-9.2.8-features-enhancements]


**Filebeat**

* Add retry back-off logic to streaming input CrowdStrike follower. [#48542](https://github.com/elastic/beats/pull/48542) 
* Add secret_state config to CEL input for encrypted storage of secrets accessible as state.secret. [#49207](https://github.com/elastic/beats/pull/49207) 

  Add a secret_state configuration field to the CEL input. When configured in a
  Fleet integration package with secret: true, the values are stored encrypted by
  Fleet. At runtime, the contents are placed at state.secret and unconditionally
  redacted in debug logs. The key &#34;secret&#34; in the plain-text state configuration
  is reserved and rejected by validation to prevent accidental unencrypted storage
  of values intended to be secret.
  
* Allow string and number arrays in httpjson chained configurations. [#49391](https://github.com/elastic/beats/pull/49391) [#16662](https://github.com/elastic/integrations/pull/16662)
* Add support for URL and URL query parsing and formatting in the Streaming input CEL environment. [#49653](https://github.com/elastic/beats/pull/49653) 

**Metricbeat**

* Add client secret authentication support to Azure App Insights module. [#48880](https://github.com/elastic/beats/pull/48880) 


### Fixes [beats-9.2.8-fixes]


**Elastic agent**

* Fix an issue that could delay reporting shutdown of Agent components. [#49414](https://github.com/elastic/beats/pull/49414) 
* Reduce AutoOps logging from info to debug for polling. [#49507](https://github.com/elastic/beats/pull/49507) 

**Filebeat**

* Fix Filestream take_over causing file re-ingestion when used with autodiscover. [#49632](https://github.com/elastic/beats/pull/49632) [#49579](https://github.com/elastic/beats/issues/49579)
* Fix compatibility of the Journald input with journald/systemd versions &lt; 242. [#49445](https://github.com/elastic/beats/pull/49445) 
* Add rate-limit backoff to CrowdStrike streaming input oauth2 transport. [#49453](https://github.com/elastic/beats/pull/49453) 

  Wrap the oauth2 HTTP transport used by the CrowdStrike falcon streaming input
  with a rate-limit-aware transport that intercepts 429 responses, reads the
  Retry-After header, and backs off before retrying. This prevents the oauth2
  token refresh from generating a burst of unauthorized requests that triggers
  CrowdStrike&#39;s 15-per-minute rate limit. The discover endpoint also returns a
  retry-after hint to the session-level retry loop as a minimum wait floor.
  
* Skip request tracer path validation when tracing is disabled to prevent input startup failures. [#49655](https://github.com/elastic/beats/pull/49655) 

  The startup path validation in cel, httpjson, http_endpoint, and entity
  analytics inputs checked whether the tracer config struct was non-nil rather
  than whether tracing was enabled. Integration package templates always include
  a tracer block (with enabled defaulting to false), so the struct is never nil.
  Under the agentless/otel runtime the relative tracer path resolves outside the
  permitted directory, causing all affected inputs to fail immediately even though
  tracing was disabled. The config-level Validate methods already used the correct
  enabled() guard; the startup paths now do the same.
  
* Fix Filebeat crash loop when running under Elastic Agent and taking too long to initialise. [#49796](https://github.com/elastic/beats/pull/49796) 

**Libbeat**

* Fixed a bug where escaped characters in syslog structured data caused an EOF error. [#49392](https://github.com/elastic/beats/pull/49392) [#43944](https://github.com/elastic/beats/issues/43944)

**Metricbeat**

* Fix unnecessary Windows filesystem metricset errors from non-existent volumes. [#49553](https://github.com/elastic/beats/pull/49553) 

  Fixes an issue where filesystem metric collection on Windows could report errors for volumes that are no longer present. Updated to gosigar v0.14.4.

**Winlogbeat**

* Skip record ID gap detection for forwarded Windows events. [#49819](https://github.com/elastic/beats/pull/49819) 

