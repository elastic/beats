## 9.4.6 [beats-release-notes-9.4.6]

### Features and enhancements [beats-9.4.6-features-enhancements]

**All**

* Update Go to 1.26.6. [#52642](https://github.com/elastic/beats/pull/52642) 

**Filebeat**

* Update `github.com/apache/thrift` to v0.24.0. [#52330](https://github.com/elastic/beats/pull/52330) 
* Add managed identity authentication to the `azure-blob-storage` input. [#52635](https://github.com/elastic/beats/pull/52635) [#47317](https://github.com/elastic/beats/issues/47317)

### Fixes [beats-9.4.6-fixes]

**All**

* Add support for index in inputs for Beat receivers. [#52662](https://github.com/elastic/beats/pull/52662) 
* Correct disk queue metrics after blocked publishes. [#52666](https://github.com/elastic/beats/pull/52666) 
* Revert non-Ironbank container images to UBI9. [#52823](https://github.com/elastic/beats/pull/52823) 

**Auditbeat**

* Fix the Auditbeat `system/package` module to skip non-Ruby Homebrew formula paths.  

**Filebeat**

* Fix `httpjson` cursor migration losing position when redirecting to CEL with a named input in Managed integration deployments.  

**Metricbeat**

* Recreate Kubernetes metadata watchers after their final metricset owner stops. [#52028](https://github.com/elastic/beats/pull/52028) [#51833](https://github.com/elastic/beats/issues/51833)
* Add configurable usage lookback and forecast windows to the Azure billing metricset. [#52209](https://github.com/elastic/beats/pull/52209) 
* Unblock a metricset whose first `fetch()` call is stuck when the wrapper shuts down. [#52619](https://github.com/elastic/beats/pull/52619) 
* Close `dbus` connections in the `system/service` and `system/users` metricsets to stop FD leaks. [#52674](https://github.com/elastic/beats/pull/52674) [#52418](https://github.com/elastic/beats/issues/52418)

**Packetbeat**

* Fix panics in Packetbeat SIP, TLS JA3, MySQL execute, and HTTP chunked parsers on malformed or truncated network input.  

**Winlogbeat**

* Fix event messages rendering zero values (for example, `PID: 0`) in place of non-string parameters.
