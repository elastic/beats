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

## 9.1.1 [beats-9.1.1-release-notes]

### Features and enhancements [beats-9.1.1-features-enhancements]

**Filebeat**

- Log CEL single object evaluation results as ECS compliant documents where possible. [45254]({{beats-issue}}45254) [45399]({{beats-pull}}45399)
- Enhanced HTTPJSON input error logging with structured error metadata conforming to Elastic Common Schema (ECS) conventions. [45653]({{beats-pull}}45653)

### Fixes [beats-9.1.1-fixes]

**Filebeat**

- Fix a panic in the winlog input that prevented it from starting. [45693]({{beats-issue}}45693) [45730]({{beats-pull}}45730)

**Metricbeat**

- Improve error messages in AWS Health [45408]({{beats-pull}}45408)
- Fix URL construction to handle query parameters properly in GET requests for Jolokia [45620]({{beats-pull}}45620)

## 9.1.0 [beats-9.1.0-release-notes]

### Features and enhancements [beats-9.1.0-features-enhancements]

**Affecting all Beats**

- Added the `now` processor, which will populate the specified target field with the current timestamp. [44795]({{beats-pull}}44795)

**Filebeat**

- Refactor & cleanup with updates to default values and documentation. [41834]({{beats-pull}}41834)
- Add support for SSL and Proxy configurations for websocket type in streaming input. [41934]({{beats-pull}}41934)
- Filestream take over now supports taking over states from other Filestream inputs and dynamic loading of inputs (autodiscover and Elastic Agent). There is a new syntax for the configuration, but the previous one can still be used. [42472]({{beats-issue}}42472) [42884]({{beats-issue}}42884) [42624]({{beats-pull}}42624)
- Refactor & cleanup with updates to default values and documentation. [41834]({{beats-pull}}41834)
- Segregated `max_workers` from `batch_size` in the GCS input. [44311]({{beats-issue}}44311) [44333]({{beats-pull}}44333)
- Add milliseconds to document timestamp from awscloudwatch Filebeat input [44306]({{beats-pull}}44306)
- Added support for specifying custom content-types and encodings in azureblobstorage input. [44330]({{beats-issue}}44330) [44402]({{beats-pull}}44402)
- Introduce lastSync start position to AWS CloudWatch input backed by state registry. [43251]({{beats-pull}}43251)
- Add proxy support to GCP Pub/Sub input. [44892]({{beats-pull}}44892)
- Segregated `max_workers` from `batch_size` in the azure-blob-storage input. [44491]({{beats-issue}}44491) [44992]({{beats-pull}}44992)
- Add support for relationship expansion to EntraID entity analytics provider. [43324]({{beats-issue}}43324) [44761]({{beats-pull}}44761)
- Update CEL mito extensions to v1.22.0. [45245]({{beats-pull}}45245)
- Add support for generalized token authentication to CEL input. [45359]({{beats-pull}}45359)

**Metricbeat**

- Add new metricset wmi for the windows module. [42017]({{beats-pull}}42017)
- Changed the Elasticsearch module behavior to only pull settings from non-system indices. [43243]({{beats-pull}}43243)
- Exclude dotted indices from settings pull in Elasticsearch module. [43306]({{beats-pull}}43306)
- Add a `jetstream` metricset to the NATS module [43310]({{beats-pull}}43310)
- Update NATS module compatibility. Oldest version supported is now 2.2.6 [43310]({{beats-pull}}43310)
- Upgrade Prometheus Library to v0.300.1. [43540]({{beats-pull}}43540)
- Add GCP Dataproc metadata collector in GCP module. [43518]({{beats-pull}}43518)
- Updated list of supported vSphere versions in the documentation. [43642]({{beats-pull}}43642)
- Add SSL support for sql module: drivers mysql, postgres, and mssql. [44748]({{beats-pull}}44748)
- Add VPN metrics to meraki module [44851]({{beats-pull}}44851)
- Add GCP cache for metadata collectors. [44432]({{beats-pull}}44432)

### Fixes [beats-9.1.0-fixes]

**Auditbeat**

- Fix potential data loss in add_session_metadata. [42795]({{beats-pull}}42795)
- auditbeat/fim: Fix FIM@ebpfevents for new kernels #44371. [44371]({{beats-pull}}44371)

**Filebeat**

- Log bad handshake details when websocket connection fails [41300]({{beats-pull}}41300)
- Fix aws region in aws-s3 input s3 polling mode.  [41572]({{beats-pull}}41572)
- Fix a logging regression that ignored to_files and logged to stdout. [44573]({{beats-pull}}44573)
- Fixed issue for "Root level readerConfig no longer respected" in azureblobstorage input. [44812]({{beats-issue}}44812) [44873]({{beats-pull}}44873)
- Fixed password authentication for ACL users in the Redis input of Filebeat. [44137]({{beats-pull}}44137)

**Heartbeat**

- Added maintenance windows support for Heartbeat. [41508]({{beats-pull}}41508)


## 9.0.4 [beats-9.0.4-release-notes]

### Features and enhancements [beats-9.0.4-features-enhancements]

**Filebeat**

- Add Fleet status updating to GCS input. [44273]({{beats-issue}}44273) [44508]({{beats-pull}}44508)
- Add Fleet status update functionality to udp input. [44419]({{beats-issue}}44419) [44785]({{beats-pull}}44785)
- Add Fleet status update functionality to tcp input. [44420]({{beats-issue}}44420) [44786]({{beats-pull}}44786)
- Add Fleet status updating to Azure Blob Storage input. [44268]({{beats-issue}}44268) [44945]({{beats-pull}}44945)
- Add Fleet status updating to HTTP JSON input. [44282]({{beats-issue}}44282) [44365]({{beats-pull}}44365)
- Add input metrics to Azure Blob Storage input. [36641]({{beats-issue}}36641) [43954]({{beats-pull}}43954)
- Add support for websocket keep_alive heartbeat in the streaming input. [42277]({{beats-issue}}42277) [44204]({{beats-pull}}44204)
- Add missing "text/csv" content-type filter support in GCS input. [44922]({{beats-issue}}44922) [44923]({{beats-pull}}44923)

**Heartbeat**

- Upgrade Node version to latest LTS v20.19.3. [45087]({{beats-pull}}45087)
- Add base64 encoding option to inline monitors. [45100]({{beats-pull}}45100)

**Metricbeat**

- Upgrade github.com/microsoft/go-mssqldb version from v1.7.2 to v1.8.2. [44990]({{beats-pull}}44990)

### Fixes [beats-9.0.4-fixes]

**Affecting all Beats**

- The Elasticsearch output now correctly applies exponential backoff when being throttled by 429s ("too many requests") from Elasticsarch. [36926]({{beats-issue}}36926) [45073]({{beats-pull}}45073)

**Winlogbeat**

- Fix EvtVarTypeAnsiString conversion. [44026]({{beats-pull}}44026)

## 9.0.3 [beats-9.0.3-release-notes]

### Features and enhancements [beats-9.0.3-features-enhancements]

**Affecting all Beats**

- Update to Go 1.24.4. [44696]({{beats-pull}}44696)

**Filebeat**

- Fix handling of ADC (Application Default Credentials) metadata server credentials in HTTPJSON input. [44349]({{beats-issue}}44349) [44436]({{beats-pull}}44436)
- Fix handling of ADC (Application Default Credentials) metadata server credentials in CEL input. [44349]({{beats-issue}}44349) [44571]({{beats-pull}}44571)
- Filestream now logs at level warn the number of files that are too small to be ingested [44751]({{beats-pull}}44751)

**Metricbeat**

- Add check for http error codes in the Metricbeat's Prometheus query submodule [44493]({{beats-pull}}44493)
- Increase default polling period for MongoDB module from 10s to 60s [44781]({{beats-pull}}44781)

### Fixes [beats-9.0.3-fixes]

**Affecting all Beats**

- Fix `dns` processor to handle IPv6 server addresses properly. [44526]({{beats-pull}}44526)
- Fix an issue where the Kafka output could get stuck if a proxied connection to the Kafka cluster was reset. [44606]({{beats-issue}}44606)
- Use Debian 11 to build linux/arm to match linux/amd64. Upgrades linux/arm64's statically linked glibc from 2.28 to 2.31. [44816]({{beats-issue}}44816)

**Filebeat**

- Handle special values of accountExpires in the Activedirectory Entity Analytics provider. [43364]({{beats-pull}}43364)
- Fix status reporting panic in GCP Pub/Sub input. [44624]({{beats-issue}}44624) [44625]({{beats-pull}}44625)
- If a Filestream input fails to be created, its ID is removed from the list of running input IDs [44697]({{beats-pull}}44697)
- Fix timeout handling by Crowdstrike streaming input. [44720]({{beats-pull}}44720)
- Ensure DEPROVISIONED Okta entities are published by Okta entityanalytics provider. [12658]({{beats-issue}}12658) [44719]({{beats-pull}}44719)
- Fix handling of cursors by the streaming input for Crowdstrike. [44364]({{beats-issue}}44364) [44548]({{beats-pull}}44548)
- Added missing "text/csv" content-type filter support in azureblobsortorage input. [44596]({{beats-issue}}44596) [44824]({{beats-pull}}44824)
- Fix unexpected EOF detection and improve memory usage. [44813]({{beats-pull}}44813)

**Heartbeat**

- Add missing dependencies to ubi9-minimal distro. [44556]({{beats-pull}}44556)

**Metricbeat**

- Fix panic in kafka consumergroup member assignment fetching when there are 0 members in consumer group. [44576]({{beats-pull}}44576)
- Sanitize error messages in Fetch method of SQL module [44577]({{beats-pull}}44577)
- Upgrade `go.mongodb.org/mongo-driver` from `v1.14.0` to `v1.17.4` to fix connection leaks in MongoDB module [44769]({{beats-pull}}44769)

## 9.0.2 [beats-9.0.2-release-notes]

### Features and enhancements [beats-9.0.2-features-enhancements]

**Affecting all Beats**

- Update Go version to v1.24.3. [44270]({{beats-pull}}44270)

**Filebeat**

- Add support for collecting device entities in the Active Directory entity analytics provider. [44309]({{beats-pull}}44309)
- The `add_cloudfoundry_metadata` processor now uses `xxhash` instead of `SHA1` for sanitizing persistent cache filenames. Existing users will experience a one-time cache invalidation as the cache store will be recreated with the new filename format. [43964]({{beats-pull}}43964)

**Metricbeat**

- Add checks for the Resty response object in all Meraki module API calls to ensure proper handling of nil responses. [44193]({{beats-pull}}44193)
- Add a latency configuration option to the Azure Monitor module. [44366]({{beats-pull}}44366)

**Osquerybeat**

- Update osquery version to v5.15.0. [43426]({{beats-pull}}43426)

### Fixes [beats-9.0.2-fixes]

**Affecting all Beats**

- Fix the 'add_cloud_metadata' processor to better support custom certificate bundles by improving how the AWS provider HTTP client is overridden. [44189]({{beats-pull}}44189)

**Auditbeat**

- Fix a potential error in the system/package component that could occur during internal package database schema migration. [44294]({{beats-issue}}44294) [44296]({{beats-pull}}44296)

**Filebeat**

- Fix endpoint path typo in the Okta entity analytics provider. [44147]({{beats-pull}}44147)
- Fix a WebSocket panic scenario that occured after exhausting the maximum number of retries. [44342]({{beats-pull}}44342)

**Metricbeat**

- Add AWS OwningAccount support for cross-account monitoring. [40570]({{beats-issue}}40570) [40691]({{beats-pull}}40691)
- Use namespace for GetListMetrics calls in AWS when available. [41022]({{beats-pull}}41022)
- Limit index stats collection to cluster-level summaries. [36019]({{beats-issue}}36019) [42901]({{beats-pull}}42901)
- Omit `tier_preference`, `creation_date` and `version` fields in output documents when not pulled from source indices. [43637]({{beats-pull}}43637)
- Add support for `_nodes/stats` URIs compatible with legacy Elasticsearch versions. [44307]({{beats-pull}}44307)

## 9.0.1 [beats-9.0.1-release-notes]

### Features and enhancements [beats-9.0.1-features-enhancements]

* For all Beats: Publish `cloud.availability_zone` by `add_cloud_metadata` processor in Azure environments. [#42601]({{beats-issue}}42601) [#43618]({{beats-pull}}43618)
* Add pagination batch size support to Entity Analytics input's Okta provider in Filebeat. [#43655]({{beats-pull}}43655)
* Update CEL mito extensions version to v1.19.0 in Filebeat. [#44098]({{beats-pull}}44098)
* Upgrade node version to latest LTS v18.20.7 in Heartbeat. [#43511]({{beats-pull}}43511)
* Add `enable_batch_api` option in Azure monitor to allow metrics collection of multiple resources using Azure batch API in Metricbeat. [#41790]({{beats-pull}}41790)

### Fixes [beats-9.0.1-fixes]

* For all Beats: Handle permission errors while collecting data from Windows services and don't interrupt the overall collection by skipping affected services. [#40765]({{beats-issue}}40765) [#43665]({{beats-pull}}43665).
* Fixed WebSocket input panic on sudden network error or server crash in Filebeat. [#44063]({{beats-issue}}44063) [44068]({{beats-pull}}44068).
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
