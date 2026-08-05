## 9.5.0 [beats-release-notes-9.5.0]



### Features and enhancements [beats-9.5.0-features-enhancements]


**All**

* Introduce `wait_for_metadata`, `wait_for_metadata_timeout`, and `wait_for_metadata_retry_period` settings to `add_kubernetes_metadata` processor to allow waiting for Kubernetes API availability before processing events. [#50509](https://github.com/elastic/beats/pull/50509) 
* Retry connecting to Docker in `add_docker_metadata` when the initial connection attempt fails. [#50595](https://github.com/elastic/beats/pull/50595) 
* Improve Beats shutdown to wait for in-flight events to be acknowledged before stopping. [#51136](https://github.com/elastic/beats/pull/51136) 

**Auditbeat**

* Add OpenTelemetry Collector receiver for Auditbeat. [#49860](https://github.com/elastic/beats/pull/49860) 
* Update quark to 0.6.0. [#51767](https://github.com/elastic/beats/pull/51767) 

**Beats**

* Add `slabqueue`, a new in-memory publisher queue that shares a single event budget across multiple pipelines while keeping per-pipeline FIFOs isolated. Selectable via queue.slab (~2.5× throughput versus `queue.mem` in stress tests). Used by default in Beat receivers to support multi-receiver intake queue sharing. [#51047](https://github.com/elastic/beats/pull/51047) 
* Turn off periodic metrics logging (`logging.metrics`) by default when a beat runs as an OTel Collector receiver. [#51711](https://github.com/elastic/beats/pull/51711) 

**Filebeat**

* Add otel_file_storage registry backend for standalone Filebeat. [#50130](https://github.com/elastic/beats/pull/50130) 

  Remove the experimental `bbolt` registry backend. Add `otel_file_storage` as a new
  registry backend for standalone Filebeat, using the same on-disk layout as the
  OpenTelemetry Collector file_storage extension.
  
* Add Jamf provider support for entity analytics minimal-state mode. [#50445](https://github.com/elastic/beats/pull/50445) 
* Add `auditd` filestream parser that populates `auditd.log.*` fields using `go-libaudit`. [#50791](https://github.com/elastic/beats/pull/50791) 
* Add `entcollect` store adapter to the Elasticsearch storage OTel extension. [#49871](https://github.com/elastic/beats/pull/49871) 
* Add an `include_file_fingerprint` configuration option to the filestream input to control `log.file.fingerprint` in events. [#50129](https://github.com/elastic/beats/pull/50129) [#50724](https://github.com/elastic/beats/issues/50724)
* Add minimal-state Active Directory entity analytics provider. [#50601](https://github.com/elastic/beats/pull/50601) [#49162](https://github.com/elastic/beats/issues/49162)
* Add minimal-state Okta entity analytics provider with bulk-fetch group enrichment. [#50685](https://github.com/elastic/beats/pull/50685) 
* Add minimal-state mode for the EntraID entity analytics provider. [#50773](https://github.com/elastic/beats/pull/50773) 
* Add Enhanced Fingerprint mode to the filestream input's `fingerprint` file identity. [#50566](https://github.com/elastic/beats/pull/50566) [#50116](https://github.com/elastic/beats/issues/50116)
* Add filestream scanner metrics to monitoring logs. [#50963](https://github.com/elastic/beats/pull/50963)
* Add Elasticsearch-backed state store support for entity-analytics Elastic Managed integration deployments. [#51210](https://github.com/elastic/beats/pull/51210)
* Reduce allocations in filestream by pooling the per-file LineReader scratch buffer. [#51197](https://github.com/elastic/beats/pull/51197) 
* Add emit macro, stream producers, and lazy JSON decode to the CEL input. [#51279](https://github.com/elastic/beats/pull/51279)
* Promote the `winlog` input to GA (generally available). [#51557](https://github.com/elastic/beats/pull/51557) 
* Add v2 `aws-s3` input with adaptive flow control. [#51598](https://github.com/elastic/beats/pull/51598)
* Add filestream harvester metrics to monitoring logs. [#51077](https://github.com/elastic/beats/pull/51077) [#36653](https://github.com/elastic/beats/issues/36653)
* Add sign-in activity enrichment to the minimal-state EntraID entity analytics provider. [#51724](https://github.com/elastic/beats/pull/51724) 
* Aggregate SQS mode health status to reduce false Degraded/Healthy transitions. [#51819](https://github.com/elastic/beats/pull/51819) [#51692](https://github.com/elastic/beats/issues/51692)
* Wire SQS health status aggregation into the v2 `aws-s3` input path. [#52001](https://github.com/elastic/beats/pull/52001) [#51692](https://github.com/elastic/beats/issues/51692)
* Add configurable User-Agent header to the CrowdStrike streaming input. [#51822](https://github.com/elastic/beats/pull/51822) 
* Reduce filestream scanner per-scan memory allocations for large idle file sets. [#51863](https://github.com/elastic/beats/pull/51863) 
* Pass configured HTTP transport to Okta minimal-state provider for TLS support. [#52062](https://github.com/elastic/beats/pull/52062) 
* Remove Authorization headers from CEL, Entity Analytics, HTTP Endpoint, and HTTP JSON input request trace logs. [#52224](https://github.com/elastic/beats/pull/52224) 

**Heartbeat**

* Add OpenTelemetry Collector receiver for Heartbeat. [#49862](https://github.com/elastic/beats/pull/49862) 

**Metricbeat**

* Add CPU number information to IIS module and Windows perfmon dataset. [#48637](https://github.com/elastic/beats/pull/48637) 
* Collect init container metrics in Kubernetes `state_container` metricset. [#50052](https://github.com/elastic/beats/pull/50052) [#49797](https://github.com/elastic/beats/issues/49797)

**Osquerybeat**

* Add RRULE scheduling and scheduled responses for Osquerybeat. [#48767](https://github.com/elastic/beats/pull/48767)
* Remove the dependency on `fslib` and implements the functionality using `go-ntfs` instead. [#49763](https://github.com/elastic/beats/pull/49763)
* Add `osquerybeatreceiver` to run Osquerybeat in the EDOT Collector. [#49868](https://github.com/elastic/beats/pull/49868) 
* Add `elastic_ntfs_partitions` and `elastic_ntfs_volumes` tables to the Osquery extension. [#50140](https://github.com/elastic/beats/pull/50140)
* Add `elastic_ntfs_file` table to the Osquery extension for MFT-based file metadata on Windows. [#50641](https://github.com/elastic/beats/pull/50641)
* Rename Osquerybeat query profiling settings to `profiling` and enable profiling by default. [#51691](https://github.com/elastic/beats/pull/51691)
* Harden Osquerybeat RRULE scheduler reload, RRULE osqueryd rendering, splay limits, and pack schedule metadata. [#48767](https://github.com/elastic/beats/pull/48767) 

**Packetbeat**

* Add OpenTelemetry Collector receiver for Packetbeat. [#49859](https://github.com/elastic/beats/pull/49859) 


### Fixes [beats-9.5.0-fixes]


**All**

* Upgrade Go to v1.26.5. [#51873](https://github.com/elastic/beats/pull/51873) 
* Fix an issue where the `worker`/`workers` setting was not respected when `loadbalance` is set to `false`. [#51041](https://github.com/elastic/beats/pull/51041) 
* Honor the enabled flag when reloading inputs. [#51472](https://github.com/elastic/beats/pull/51472) 

**Beats**

* Fix an issue that could cause some beats to delay longer than needed on shutdown. [#49482](https://github.com/elastic/beats/pull/49482) 

**Elastic agent**

* Fix Beat receiver shutdown hanging when start fails before the beater is launched. [#52106](https://github.com/elastic/beats/pull/52106) [#52009](https://github.com/elastic/beats/issues/52009)

**Filebeat**

* Harden Salesforce input batching compatibility and auth-failure recovery. [#50149](https://github.com/elastic/beats/pull/50149)
* Fix Okta minimal-state provider to accept configs with both `okta_token` and `oauth2`. [#51078](https://github.com/elastic/beats/pull/51078) [#51005](https://github.com/elastic/beats/issues/51005)
* Fix S3 polling input to exclude same-bucket backup objects from listing. [#51912](https://github.com/elastic/beats/pull/51912)
* Honor `path_style` for non-AWS S3 buckets in the `aws-s3` input. [#52003](https://github.com/elastic/beats/pull/52003)
* Use a stable status message for SQS receive errors to prevent update churn during sustained outages. [#52059](https://github.com/elastic/beats/pull/52059) 
* Do not mark the CEL input's health as degraded when the maximum number of executions is exceeded. [#52060](https://github.com/elastic/beats/pull/52060) 
* Honor the configured language when rendering Windows events. [#52094](https://github.com/elastic/beats/pull/52094) [#7332](https://github.com/elastic/sdh-beats/issues/7332)
* Fix the `aws-s3` input silently dropping events when a parser clears the message content. [#52077](https://github.com/elastic/beats/pull/52077) 
* Fix `journald` input facility filters on `journalctl` older than 245. [#52231](https://github.com/elastic/beats/pull/52231) 
* The `journald` input now reports a Degraded status when `journalctl` repeatedly exits without delivering any data, instead of staying Healthy while collecting no events. [#52232](https://github.com/elastic/beats/pull/52232) 
* Fix TCP and UDP inputs hanging during shutdown. [#52292](https://github.com/elastic/beats/pull/52292) 

**Heartbeat**

* Fix panic with multiple Heartbeat receivers. [#52010](https://github.com/elastic/beats/pull/52010) 

**Libbeat**

* Add LDAP channel binding for Windows SSPI binds. [#49733](https://github.com/elastic/beats/pull/49733) 
* Fix `add_host_metadata` `host.geo` map sharing across events causing data corruption. [#49722](https://github.com/elastic/beats/pull/49722) [#49721](https://github.com/elastic/beats/issues/49721)

**Osquerybeat**

* Reject CPIO entries in macOS .pkg payloads whose paths escape the destination directory, preventing path traversal during extraction (CWE-22). [#51446](https://github.com/elastic/beats/pull/51446) 
* Propagate Osquery live-query space IDs to action response events. [#50808](https://github.com/elastic/beats/pull/50808) 
* Fix the serialization of ads and active columns in `elastic_ntfs_file` table to use boolean instead of integer. [#51629](https://github.com/elastic/beats/pull/51629) 
* Accept Osquery packs mixing interval-scheduled and unscheduled queries again. [#52242](https://github.com/elastic/beats/pull/52242) [#51450](https://github.com/elastic/beats/issues/51450)

**Winlogbeat**

* Fix Kerberos ticket status code descriptions not being set in the Security event log ingest pipeline when `winlog.event_data.Status` arrives in lower case. [#51681](https://github.com/elastic/beats/pull/51681) 

