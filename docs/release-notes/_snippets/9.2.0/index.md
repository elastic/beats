## 9.2.0 [beats-release-notes-9.2.0]

_This release also includes: [Breaking changes](/release-notes/breaking-changes.md#beats-9.2.0-breaking-changes)._


### Features and enhancements [beats-9.2.0-features-enhancements]


**All**

* The following output latency_delta metrics are now included when `logging.metrics` is enabled: `output.latency_delta.{count, max, median, min, p99}`. This only includes data since the last internal metrics was logged. [#45749](https://github.com/elastic/beats/pull/45749) 

**Auditbeat**

* Add new ETW FIM backend for Windows. [#45887](https://github.com/elastic/beats/pull/45887) 

**Filebeat**

* TCP and UDP inputs now support multiple pipeline workers configured via `number_of_workers`. Increasing the number of workers improves performance when slow processors are used by decoupling reading from the network connection and publishing. [#45124](https://github.com/elastic/beats/pull/45124) [#43674](https://github.com/elastic/beats/issues/43674)
* Add beta GZIP file ingestion in filestream. [#45301](https://github.com/elastic/beats/pull/45301) 
* Updated the `parse_aws_vpc_flow_log` processor to handle AWS VPC flow log v6, v7, and v8 fields. [#45746](https://github.com/elastic/beats/pull/45746) 
* Add OAuth2 support for Okta provider in Entity Analytics input. [#45753](https://github.com/elastic/beats/pull/45753) 
* Improve error reporting for schemeless URLs in HTTP JSON input. [#45953](https://github.com/elastic/beats/pull/45953) 
* Add `remaining_executions` global to the CEL input evaluation context. [#46210](https://github.com/elastic/beats/pull/46210) 
* Journald input now supports reading from multiple journals, including remote ones. [#46722](https://github.com/elastic/beats/pull/46722) [#46656](https://github.com/elastic/beats/issues/46656)

**Metricbeat**

* Improve the Prometheus helper to handle multiple content types including blank and invalid headers. [#47085](https://github.com/elastic/beats/pull/47085) 

**Osquerybeat**

* Upgrade osquery version to 5.18.1. [#46624](https://github.com/elastic/beats/pull/46624) 

**Packetbeat**

* Bump Windows Npcap version to v1.83. [#46809](https://github.com/elastic/beats/pull/46809) 


### Fixes [beats-9.2.0-fixes]


**All**

* Make data updates in `add_host_metadata` processor synchronous. [#46546](https://github.com/elastic/beats/pull/46546) 
* Prevent panic in logstash output when trying to send events while shutting down. [#46960](https://github.com/elastic/beats/pull/46960) [#46889](https://github.com/elastic/beats/issues/46889)
* Prevent panic in replace processor for non-string values. [#47009](https://github.com/elastic/beats/pull/47009) [#42308](https://github.com/elastic/beats/issues/42308)
* Autodiscover now correctly updates Kubernetes metadata on node and pod label changes. [#47034](https://github.com/elastic/beats/pull/47034) [#46979](https://github.com/elastic/beats/issues/46979)
* Prevent 3s startup delay when add_cloud_metadata is used with debug logs. [#47058](https://github.com/elastic/beats/pull/47058) [#44203](https://github.com/elastic/beats/issues/44203)
* Update elastic-agent-system-metrics version to v0.13.3. [#47104](https://github.com/elastic/beats/pull/47104) [#47054](https://github.com/elastic/beats/issues/47054)

  Removes &#34;Accurate CPU counts not available on platform&#34; log spam at the debug log level.
* Allows users to customize their data stream namespace to &#34;generic&#34;. [#47140](https://github.com/elastic/beats/pull/47140) 

**Filebeat**

* Fix defer usage for stopped status reporting. [#46916](https://github.com/elastic/beats/pull/46916) 

**Metricbeat**

* Fix missing AWS cloudwatch metrics with linked accounts and same dimensions. [#46978](https://github.com/elastic/beats/pull/46978) [#15362](https://github.com/elastic/integrations/issues/15362)
* Add a fix to handle blank content-type headers in HTTP responses for Prometheus. [#47027](https://github.com/elastic/beats/pull/47027) 
* Add pagination support to the device health metricset in the meraki module. [#46938](https://github.com/elastic/beats/pull/46938) [#15551](https://github.com/elastic/integrations/issues/15551)

