---
navigation_title: Beats
mapped_pages:
  - https://www.elastic.co/guide/en/beats/libbeat/current/release-notes.html
---

# {{beats}} release notes [beats-release-notes]
Review the changes, fixes, and more in each version of {{beats}}.

To check for security updates, go to [Security announcements for the Elastic stack](https://discuss.elastic.co/c/announcements/security-announcements/31).

% Release notes include only features, enhancements, and fixes. Add breaking changes, deprecations, and known issues to the applicable release notes sections.

% ## version.next [beats-versionext-release-notes]

% ### Features and enhancements [beats-versionext-features-enhancements]

% ### Fixes [beats-versionext-fixes]

## 9.0.1 [beats-9.0.1-release-notes]

### Features and enhancements [beats-9.0.1-features-enhancements]

* For all Beats: Publish `cloud.availability_zone` by `add_cloud_metadata` processor in Azure environments. [#42601]({{beats-issue}}42601) [#43618]({{beats-pull}}43618)
* Add pagination batch size support to Entity Analytics input's Okta provider in Filebeat. [#43655]({{beats-pull}}43655)
* Update CEL mito extensions version to v1.19.0 in Filebeat. [#44098]({{beats-pull}}44098)
* Upgrade node version to latest LTS v18.20.7 in Heartbeat. [#43511]({{beats-pull}}43511)
* Add `enable_batch_api` option in Azure monitor to allow metrics collection of multiple resources using Azure batch API in Metricbeat. [#41790]({{beats-pull}}41790)

### Fixes [beats-9.0.1-fixes]

* For all Beats: Handle permission errors while collecting data from Windows services and don't interrupt the overall collection by skipping affected services. [#40765]({{beats-issue}}40765) [#43665]({{beats-pull}}43665).
* Fixed websocket input panic on sudden network error or server crash in Filebeat. [#44063]({{beats-issue}}44063) [44068]({{beats-pull}}44068).
* [Filestream] Log the "reader closed" message on the debug level to avoid log spam in Filebeat. [#44051]({{beats-pull}}44051)
* Fix links to CEL mito extension functions in input documentation in Filebeat. [#44098]({{beats-pull}}44098)

## 9.0.0 [beats-900-release-notes]

### Features and enhancements [beats-900-features-enhancements]
* Improves logging in system/socket in Auditbeat. [#41571]({{beats-pull}}41571)
* Adds out of the box support for Amazon EventBridge notifications over SQS to S3 input in Filebeat. [#40006]({{beats-pull}}40006)
* Update CEL mito extensions to v1.16.0 in Filebeat. [#41727]({{beats-pull}}41727)
* Filebeat's registry is now added to the Elastic-Agent diagnostics bundle. [#33238]({{beats-issue}}33238) and [#41795]({{beats-pull}}41795)
* Adds `unifiedlogs` input for MacOS in Filebeat. [#41791]({{beats-pull}}41791)
* Adds evaluation state dump debugging option to CEL input in Filebeat. [#41335]({{beats-pull}}41335)
* Rate limiting operability improvements in the Okta provider of the Entity Analytics input in Filebeat. [#40106]({{beats-issue}}40106) and [#41977]({{beats-pull}}41977)
* Rate limiting fault tolerance improvements in the Okta provider of the Entity Analytics input in Filebeat. [#40106]({{beats-issue}}40106) [#42094]({{beats-pull}}42094)
* Introduces ignore older and start timestamp filters for AWS S3 input in Filebeat. [#41804]({{beats-pull}}41804)
* Journald input now can report its status to Elastic-Agent in Filebeat. [#39791]({{beats-issue}}39791) and [#42462]({{beats-pull}}42462)
* Publish events progressively in the Okta provider of the Entity Analytics input in Filebeat. [#40106]({{beats-issue}}40106) and [#42567]({{beats-pull}}42567)
* Journald `include_matches.match` now accepts `+` to represent a logical disjunction (OR) in Filebeat. [#40185]({{beats-issue}}40185) and #[42517]({{beats-pull}}42517)
* The journald input is now generally available in Filebeat. [#42107]({{beats-pull}}42107)
* Adds support for RFC7231 methods to HTTP monitors in Heartbeat. [#41975]({{beats-pull}}41975)
* Adds `use_kubeadm` config option in kubernetes module in order to toggle kubeadm-config API requests in Metricbeat. [#40086]({{beats-pull}}40086)
* Preserve queries for debugging when `merge_results: true` in SQL module in Metricbeat. [#42271]({{beats-pull}}42271)
* Collect more fields from ES node/stats metrics and only those that are necessary in Metricbeat. [#42421]({{beats-pull}}42421)
* Adds benchmark module in Metricbeat. [#41801]({{beats-pull}}41801)
* Increase maximum query timeout to 24 hours in Osquerybeat. [42356]({{beats-pull}}42356)
* Properly set events `UserData` when experimental API is used in Winlogbeat. [#41525]({{beats-pull}}41525)
* Include XML is respected for experimental API in Winlogbeat. [#41525]({{beats-pull}}41525)
* Forwarded events use renderedtext info for experimental API in Winlogbeat. [#41525]({{beats-pull}}41525)
* Language setting is respected for experimental API in Winlogbeat. [#41525]({{beats-pull}}41525)
* Language setting also added to decode XML wineventlog processor in Winlogbeat. [#41525]({{beats-pull}}41525)
* Format embedded messages in the experimental API in Winlogbeat. [#41525]({{beats-pull}}41525)
* Make the experimental API GA and rename it to winlogbeat-raw in Winlogbeat. [#39580]({{beats-issue}}39580) and [#41770]({{beats-pull}}41770)
* Removes 22 clause limitation in Winlogbeat. [#35047]({{beats-issue}}35047) and [#42187]({{beats-pull}}42187)
* Adds handling for recoverable publisher disabled errorsin Winlogbeat. [#35316]({{beats-issue}}35316) and [#42187]({{beats-pull}}42187)
* Removes Functionbeat binaries from CI pipelines. [#40745]({{beats-issue}}40745) and [#41506]({{beats-pull}}41506)
* Update Go version to 1.24.0. [#42705]({{beats-pull}}42705)
* Add `etw` input fallback to attach an already existing session in Filebeat. [#42847]({{beats-pull}}42847)
* Update CEL mito extensions to v1.17.0 in Filebeat. [#42851]({{beats-pull}}42851)
* Winlog input  in Filebeat cam now report its status to Elastic Agent. [#43089]({{beats-pull}}43089)
* Add configuration option to limit HTTP Endpoint body size in Filebeat. [#43171]({{beats-pull}}43171)
* Add a new `match_by_parent_instance` option to `perfmon` module in Metricbeat. [#43002]({{beats-pull}}43002)
* Add a warning log to `metricbeat.vsphere` in Metricbeat in case vSphere connection has been configured as insecure. [#43104]({{beats-pull}}43104)

### Fixes [beats-900-fixes]
* hasher: Add a cached hasher for upcoming backend in Auditbeat. [#41952]({{beats-pull}}41952)
* Split common tty definitions in Auditbeat. [#42004]({{beats-pull}}42004)
* Redact authorization headers in HTTPJSON debug logs in Filebeat. [#41920]({{beats-pull}}41920)
* Further rate limiting fix in the Okta provider of the Entity Analytics input in Filebeat. [#40106]({{beats-issue}}40106) and [#41977]({{beats-pull}}41977)
* The `_id` generation process for S3 events has been updated to incorporate the LastModified field. This enhancement ensures that the `_id` is unique in Filebeat. [#42078]({{beats-pull}}42078)
* Fixes truncation of bodies in request tracing by limiting bodies to 10% of the maximum file size in Filebeat. [#42327]({{beats-pull}}42327)
* [Journald] Fixes handling of `journalctl` restart. A known symptom was broken multiline messages when there was a restart of journalctl while aggregating the lines in Filebeat. [#41331]({{beats-issue}}41331) and [#42595]({{beats-pull}}42595)
* Fixwa bug where Metricbeat unintentionally triggers Windows ASR in Metricbeat. [#42177]({{beats-pull}}42177)
* Removes `hostname` field from ZooKeeper's `mntr` data stream in Metricbeat. [41887]({{beats-pull}}41887)
* Properly marshal nested structs in ECS fields, fixing issues with mixed cases in field names in Packetbeat. [42116]({{beats-pull}}42116)
* Fixed race conditions in the global ratelimit processor that could drop events or apply rate limiting incorrectly in Filebeat. [42966]({{beats-pull}}42966)
* Prevent computer details being returned for user queries by Activedirectory Entity Analytics provider in Filebeat. [#11818]({{beats-issue}}11818) and [#42796]({{beats-pull}}42796)
* Handle unexpected EOF error in aws-s3 input and enforce retrying using download failed error in Filebeat. [#42420]({{beats-pull}}42420)
* Prevent azureblobstorage input from logging key details during blob fetch operations in Filebeat. [#43169]({{beats-pull}}43169)
* Add AWS OwningAccount support for cross account monitoring in Metricbeat. [#40570]({{beats-issue}}40570) and [#40691]({{beats-pull}}40691)
* Fix logging argument number mismatch in Metricbeat(Redis). [#43072]({{beats-pull}}43072)
* Reset EventLog if error EOF is encountered in Winlogbeat. [#42826]({{beats-pull}}42826)
* Implement backoff on error retrial in Winlogbeat. [#42826]({{beats-pull}}42826)
* Fix boolean key in security pipelines and sync pipelines with integration in Winlogbeat. [#43027]({{beats-pull}}43027)
