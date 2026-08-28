## 9.4.6 [beats-release-notes-9.4.6]



### Features and enhancements [beats-9.4.6-features-enhancements]


**All**

* Update Go to 1.26.6. [#52642](https://github.com/elastic/beats/pull/52642) 

**Filebeat**

* Update github.com/apache/thrift to v0.24.0. [#52330](https://github.com/elastic/beats/pull/52330) 
* Add managed identity authentication to the azure-blob-storage input. [#52635](https://github.com/elastic/beats/pull/52635) [#47317](https://github.com/elastic/beats/issues/47317)

  The azure-blob-storage input can now authenticate with the managed identity of the Azure host that runs Filebeat. Set auth.managed_identity.enabled to true to use the system-assigned identity of the host, and set auth.managed_identity.client_id to select a user-assigned identity. The configuration then holds no account key, connection string or client secret.


### Fixes [beats-9.4.6-fixes]


**All**

* Add support for index in inputs for beatreceivers. [#52662](https://github.com/elastic/beats/pull/52662) 
* Correct disk queue metrics after blocked publishes. [#52666](https://github.com/elastic/beats/pull/52666) 
* Revert non-Ironbank container images to UBI9. [#52823](https://github.com/elastic/beats/pull/52823) 

**Auditbeat**

* Fix auditbeat system/package module to skip non-Ruby Homebrew formula paths.  

**Filebeat**

* Fix httpjson cursor migration losing position when redirecting to CEL with a named input in Agentless deployments.  

  When an httpjson input with an id field uses run_as_cel: true to redirect to
  the CEL input, the cursor migration path was not calling SetID on the state
  store before reading the stored cursor. In Agentless deployments backed by the
  Elasticsearch state store, SetID selects the per-input index, so omitting the
  call caused the migration to read from the wrong index and silently fall back
  to the default cursor state.
  

**Metricbeat**

* Recreate Kubernetes metadata watchers after their final metricset owner stops. [#52028](https://github.com/elastic/beats/pull/52028) [#51833](https://github.com/elastic/beats/issues/51833)
* Add configurable usage lookback and forecast windows to the Azure billing metricset. [#52209](https://github.com/elastic/beats/pull/52209) 

  Adds `billing_usage_lookback` and `billing_forecast_window` duration config options to the
  Azure billing metricset. `billing_usage_lookback` (default `24h`) allows re-querying multiple
  previous days of usage data so that Azure&#39;s documented up-to-72h cost corrections are captured.
  `billing_forecast_window` (default `720h`) controls the length of the forecast period. Defaults
  preserve the existing behaviour.
  
* Unblock a metricset whose first Fetch() call is stuck when the wrapper shuts down. [#52619](https://github.com/elastic/beats/pull/52619) 
* Close dbus connections in system service/users metricsets to stop FD leaks. [#52674](https://github.com/elastic/beats/pull/52674) [#52418](https://github.com/elastic/beats/issues/52418)

**Packetbeat**

* Fix panics in packetbeat SIP, TLS JA3, MySQL execute, and HTTP chunked parsers on malformed or truncated network input.  

**Winlogbeat**

* Fix event messages rendering zero values (e.g. &#34;PID: 0&#34;) in place of non-string parameters.  

