---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/libbeat/8.17/release-notes.html
  - https://www.elastic.co/guide/en/beats/libbeat/current/release-notes.html
---

# {{beats}} release notes [beats-release-notes]
Review the changes, fixes, and more in each version of {{beats}}.

To check for security updates, go to [Security announcements for the Elastic stack](https://discuss.elastic.co/c/announcements/security-announcements/31).

% Release notes include only features, enhancements, and fixes. Add breaking changes, deprecations, and known issues to the applicable release notes sections.

% ## version.next [beats-versionext-release-notes]

% ### Features and enhancements [beats-versionext-features-enhancements]

% ### Fixes [beats-versionext-fixes]

## 9.0.0 [beats-900-release-notes]

### Features and enhancements [beats-900-features-enhancements]
* Improves logging in system/socket in Auditbeat {pull}41571[41571]
* Adds out of the box support for Amazon EventBridge notifications over SQS to S3 input in Filebeat {pull}40006[40006]
* Update CEL mito extensions to v1.16.0 in Filebeat {pull}41727[41727]
* Filebeat's registry is now added to the Elastic-Agent diagnostics bundle {issue}33238[33238] and {pull}41795[41795]
* Adds `unifiedlogs` input for MacOS in Filebeat {pull}41791[41791]
* Adds evaluation state dump debugging option to CEL input in Filebeat {pull}41335[41335]
* Rate limiting operability improvements in the Okta provider of the Entity Analytics input in Filebeat {issue}40106[40106] and {pull}41977[41977]
* Rate limiting fault tolerance improvements in the Okta provider of the Entity Analytics input in Filebeat {issue}40106[40106] {pull}42094[42094]
* Introduces ignore older and start timestamp filters for AWS S3 input in Filebeat {pull}41804[41804]
* Journald input now can report its status to Elastic-Agent in Filebeat {issue}39791[39791] and {pull}42462[42462]
* Publish events progressively in the Okta provider of the Entity Analytics input in Filebeat {issue}40106[40106] and {pull}42567[42567]
* Journald `include_matches.match` now accepts `+` to represent a logical disjunction (OR) in Filebeat {issue}40185[40185] and {pull}42517[42517]
* The journald input is now generally available in Filebeat {pull}42107[42107]
* Adds support for RFC7231 methods to HTTP monitors in Heartbeat {pull}41975[41975]
* Adds `use_kubeadm` config option in kubernetes module in order to toggle kubeadm-config API requests in Metricbeat {pull}40086[40086]
* Preserve queries for debugging when `merge_results: true` in SQL module in Metricbeat {pull}42271[42271]
* Collect more fields from ES node/stats metrics and only those that are necessary in Metricbeat {pull}42421[42421]
* Adds benchmark module in Metricbeat {pull}41801[41801]
* Increase maximum query timeout to 24 hours in Osquerybeat {pull}42356[42356]
* Properly set events `UserData` when experimental API is used in Winlogbeat {pull}41525[41525]
* Include XML is respected for experimental API in Winlogbeat {pull}41525[41525]
* Forwarded events use renderedtext info for experimental API in Winlogbeat {pull}41525[41525]
* Language setting is respected for experimental API in Winlogbeat {pull}41525[41525]
* Language setting also added to decode XML wineventlog processor in Winlogbeat {pull}41525[41525]
* Format embedded messages in the experimental API in Winlogbeat {pull}41525[41525]
* Make the experimental API GA and rename it to winlogbeat-raw in Winlogbeat {issue}39580[39580] and {pull}41770[41770]
* Removes 22 clause limitation in Winlogbeat {issue}35047[35047] and {pull}42187[42187]
* Adds handling for recoverable publisher disabled errorsin Winlogbeat {issue}35316[35316] and {pull}42187[42187]
* Removes Functionbeat binaries from CI pipelines {issue}40745[40745] and {pull}41506[41506]

### Fixes [beats-900-fixes]
* hasher: Add a cached hasher for upcoming backend in Auditbeat {pull}41952[41952]
* Split common tty definitions in Auditbeat {pull}42004[42004]
* Redact authorization headers in HTTPJSON debug logs in Filebeat {pull}41920[41920]
* Further rate limiting fix in the Okta provider of the Entity Analytics input in Filebeat {issue}40106[40106] and {pull}41977[41977]
* The `_id` generation process for S3 events has been updated to incorporate the LastModified field. This enhancement ensures that the `_id` is unique in Filebeat {pull}42078[42078]
* Fixes truncation of bodies in request tracing by limiting bodies to 10% of the maximum file size in Filebeat {pull}42327[42327]
* [Journald] Fixes handling of `journalctl` restart. A known symptom was broken multiline messages when there was a restart of journalctl while aggregating the lines in Filebeat {issue}41331[41331] and {pull}42595[42595]
* Fixwa bug where Metricbeat unintentionally triggers Windows ASR in Metricbeat {pull}42177[42177]
* Removes `hostname` field from ZooKeeper's `mntr` data stream in Metricbeat {pull}41887[41887]
* Properly marshal nested structs in ECS fields, fixing issues with mixed cases in field names in Packetbeat {pull}42116[42116]