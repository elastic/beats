## 9.5.0 [beats-release-notes-9.5.0]



### Features and enhancements [beats-9.5.0-features-enhancements]


**All**

* Introduce `wait_for_metadata` and `wait_for_metadata_timeout` and `wait_for_metadata_retry_period` settings to `add_kubernetes_metadata` processor to allow waiting for Kubernetes API availability before processing events. [#50509](https://github.com/elastic/beats/pull/50509) 
* Retry connecting to Docker in `add_docker_metadata` when the initial connection attempt fails. [#50595](https://github.com/elastic/beats/pull/50595) 

  The `add_docker_metadata` processor now retries connecting to Docker at
  startup until `wait_for_metadata_timeout` expires. The timeout defaults to
  30s and includes the initial connection attempt. Set `wait_for_metadata` to
  true to block startup until Docker metadata is available. Set
  `wait_for_metadata_timeout` to 0 to retry indefinitely. Configure retry
  cadence with `wait_for_metadata_retry_period`.
  
* Improve Beats shutdown to wait for in-flight events to be acknowledged before stopping. [#51136](https://github.com/elastic/beats/pull/51136) 

**Auditbeat**

* Add OpenTelemetry Collector receiver for Auditbeat. [#49860](https://github.com/elastic/beats/pull/49860) 
* Update quark to 0.6.0. [#51767](https://github.com/elastic/beats/pull/51767) 

**Beats**

* Add slabqueue, a new in-memory publisher queue that shares a single event budget across multiple pipelines while keeping per-pipeline FIFOs isolated. Selectable via queue.slab (~2.5× throughput vs queue.mem in stress tests); used by default in Beat receivers to support multi-receiver intake queue sharing. [#51047](https://github.com/elastic/beats/pull/51047) 
* Disable periodic metrics logging (`logging.metrics`) by default when a beat runs as an OTel Collector receiver. [#51711](https://github.com/elastic/beats/pull/51711) 

**Filebeat**

* Add otel_file_storage registry backend for standalone Filebeat. [#50130](https://github.com/elastic/beats/pull/50130) 

  Remove the experimental `bbolt` registry backend. Add `otel_file_storage` as a new
  registry backend for standalone Filebeat, using the same on-disk layout as the
  OpenTelemetry Collector file_storage extension.
  
* Add Jamf provider support for entity analytics minimal-state mode. [#50445](https://github.com/elastic/beats/pull/50445) 
* Add auditd filestream parser that populates auditd.log.* fields using go-libaudit. [#50791](https://github.com/elastic/beats/pull/50791) 
* Add entcollect adapter for entity analytics minimal state providers. [#49871](https://github.com/elastic/beats/pull/49871) 

  Add adapter infrastructure to bridge the entcollect library into the
  entity analytics input. This includes a kvstore transaction-to-store
  adapter, a document-to-beat.Event publisher closure, a minimal state
  provider registry, and a generic sync loop with timer, buffering, and
  ACK-then-commit semantics.
  
* Add entcollect store adapter to the Elasticsearch storage OTel extension. [#49871](https://github.com/elastic/beats/pull/49871) 
* Filestream: include_file_fingerprint controls log.file.fingerprint in events. [#50129](https://github.com/elastic/beats/pull/50129) [#50724](https://github.com/elastic/beats/issues/50724)

  Add an `include_file_fingerprint` option (default: `true`) to the filestream
  input. By default `log.file.fingerprint` is included in published events when
  `file_identity.fingerprint` is configured, preserving the previous behaviour.
  Set `include_file_fingerprint: false` to omit the field and reduce network,
  indexing, and storage costs at scale. A still-growing fingerprint is never
  added to events; only the completed SHA-256 is published.
  
* Add minimal-state Active Directory entity analytics provider. [#50601](https://github.com/elastic/beats/pull/50601) [#49162](https://github.com/elastic/beats/issues/49162)
* Add minimal-state Okta entity analytics provider with bulk-fetch group enrichment. [#50685](https://github.com/elastic/beats/pull/50685) 
* Add minimal-state mode for the EntraID entity analytics provider. [#50773](https://github.com/elastic/beats/pull/50773) 

  The EntraID (Azure AD) entity analytics provider now supports
  use_minimal_state: true, replacing persistent entity and group graph
  storage with delta-query-based sync and in-memory transitive group
  membership computation.
  
* Add Enhanced Fingerprint mode to the filestream input&#39;s `fingerprint` file identity.
When enabled (`file_identity.fingerprint.growing: true`, the default), files smaller
than `prospector.scanner.fingerprint.offset &#43; prospector.scanner.fingerprint.length`
are tracked instead of being skipped as too small. No data duplication happens
on upgrade or when enabling the enhanced fingerprint. The new behaviour is
opt-out (`growing: false` restores the legacy fingerprint behaviour).
. [#50566](https://github.com/elastic/beats/pull/50566) [#50116](https://github.com/elastic/beats/issues/50116)
* Add Filestream scanner metrics to monitoring logs. [#50963](https://github.com/elastic/beats/pull/50963) 

  The following Filestream metrics are added to the logs and the `/stats` HTTP endpoint:
   - files_empty
   - files_ignored
   - files_matched
   - files_no_ingest_target
   - files_unique
  
  The metrics are gauges aggregated from all running Filestream inputs
  
* Add Elasticsearch-backed state store support for entity-analytics agentless deployments. [#51210](https://github.com/elastic/beats/pull/51210) 

  The entity-analytics minimal-state input can now use an Elasticsearch-backed
  state store when running in agentless mode. State is persisted to an
  Elasticsearch index (agentless-state-&lt;input-id&gt;) instead of local bbolt,
  enabling stateless pod scheduling. The store backend is selected automatically
  based on the AGENTLESS_ELASTICSEARCH_STATE_STORE_INPUT_TYPES environment
  variable set by the agentless controller.
  
* Reduce allocations in filestream by pooling the per-file LineReader scratch buffer. [#51197](https://github.com/elastic/beats/pull/51197) 
* Add emit macro, stream producers, and lazy JSON decode to the CEL input. [#51279](https://github.com/elastic/beats/pull/51279) 

  Add the emit macro for publishing events during CEL evaluation instead of
  collecting them in the state.events array. Combined with stream_gzip,
  stream_zip, and decode_json_stream_lazy, this enables streaming
  decompression and decoding of large payloads without holding all records
  in memory.
  
* Promote the `winlog` input to GA (generally available). [#51557](https://github.com/elastic/beats/pull/51557) 
* Add v2 aws-s3 input with adaptive flow control. [#51598](https://github.com/elastic/beats/pull/51598) 

  The aws-s3 input has been rewritten with a simpler architecture, adaptive
  concurrency control, and unified state management. The new implementation can
  be enabled by setting features.aws_s3_v2.enabled: true in the beat
  configuration.
  
* Add Filestream harvester metrics to monitoring logs. [#51077](https://github.com/elastic/beats/pull/51077) [#36653](https://github.com/elastic/beats/issues/36653)

  The following Filestream metrics are added to the logs and the `/stats` HTTP endpoint:
   - files_ingested_percent_100
   - files_ingested_percent_95_99
   - files_ingested_percent_lt_95
  
  These gauges are aggregated across all running Filestream inputs and
  count active plain-file harvesters only.
  
* Add sign-in activity enrichment to the minimal-state EntraID entity analytics provider. [#51724](https://github.com/elastic/beats/pull/51724) 
* Aggregate SQS mode health status to reduce false Degraded/Healthy transitions. [#51819](https://github.com/elastic/beats/pull/51819) [#51692](https://github.com/elastic/beats/issues/51692)

  Replace scattered per-event status reporting in the aws-s3 SQS input with a
  centralized health aggregator. The input now stays Running while making forward
  progress despite bounded retryable failures, reports Degraded only for sustained
  conditions needing operator action (persistent receive failures, delete/finalize
  errors, poison-pill messages), filters out context.Canceled from shutdown/reload,
  and clears Degraded only when the specific causing condition resolves.
  
* Wire SQS health status aggregation into the v2 aws-s3 input path. [#52001](https://github.com/elastic/beats/pull/52001) [#51692](https://github.com/elastic/beats/issues/51692)

  Apply the sqsHealth aggregator (from the legacy SQS path) to the v2 SQS
  input. The v2 path now has the same health reporting semantics: transient
  failures stay Running, sustained failures degrade with condition-specific
  messages, and context cancellation from shutdown is suppressed.
  
* Add configurable User-Agent header to the CrowdStrike streaming input. [#51822](https://github.com/elastic/beats/pull/51822) 
* Reduce filestream scanner per-scan memory allocations for large idle file sets. [#51863](https://github.com/elastic/beats/pull/51863) 
* Pass configured HTTP transport to Okta minimal-state provider for TLS support. [#52062](https://github.com/elastic/beats/pull/52062) 
* Remove Authorization headers from CEL, Entity Analytics, HTTP Endpoint, and HTTP JSON input request trace logs. [#52224](https://github.com/elastic/beats/pull/52224) 

**Heartbeat**

* Add OpenTelemetry Collector receiver for Heartbeat. [#49862](https://github.com/elastic/beats/pull/49862) 

**Metricbeat**

* Adds cpu number information to iis module and windows perfmon dataset. [#48637](https://github.com/elastic/beats/pull/48637) 
* Collect init container metrics in kubernetes state_container metricset. [#50052](https://github.com/elastic/beats/pull/50052) [#49797](https://github.com/elastic/beats/issues/49797)

**Osquerybeat**

* Add RRULE scheduling and scheduled responses for osquerybeat. [#48767](https://github.com/elastic/beats/pull/48767) 
* Removes the dependency on fslib and implements the functionality using go-ntfs instead. [#49763](https://github.com/elastic/beats/pull/49763) 
* Add osquerybeatreceiver to run osquerybeat under the EDOT collector. [#49868](https://github.com/elastic/beats/pull/49868) 
* Add elastic_ntfs_partitions and elastic_ntfs_volumes tables to the osquery extension. [#50140](https://github.com/elastic/beats/pull/50140) 
* Add elastic_ntfs_file table to the osquery extension for MFT-based file metadata on Windows. [#50641](https://github.com/elastic/beats/pull/50641) 
* Rename osquerybeat query profiling settings to `profiling` and enable profiling by default. [#51691](https://github.com/elastic/beats/pull/51691) 
* Harden osquerybeat RRULE scheduler reload, RRULE osqueryd rendering, splay limits, and pack schedule metadata. [#48767](https://github.com/elastic/beats/pull/48767) 

**Packetbeat**

* Add OpenTelemetry Collector receiver for Packetbeat. [#49859](https://github.com/elastic/beats/pull/49859) 


### Fixes [beats-9.5.0-fixes]


**All**

* Upgrade to go 1.26.5. [#51873](https://github.com/elastic/beats/pull/51873) 
* This fixes a bug where worker/workers setting was not respected when loadbalance is set to false. [#51041](https://github.com/elastic/beats/pull/51041) 
* Honor the enabled flag when reloading inputs. [#51472](https://github.com/elastic/beats/pull/51472) 

**Beats**

* Fix an issue that could cause some beats to delay longer than needed on shutdown. [#49482](https://github.com/elastic/beats/pull/49482) 

**Elastic agent**

* Fix beat receiver Shutdown hanging when Start fails before the beater is launched. [#52106](https://github.com/elastic/beats/pull/52106) [#52009](https://github.com/elastic/beats/issues/52009)

**Filebeat**

* Harden Salesforce input batching compatibility and auth-failure recovery. [#50149](https://github.com/elastic/beats/pull/50149) 

  Preserve Salesforce object cursor continuity across bounded-batch upgrades
  and toggles. Existing installs now seed batched windows from legacy
  first_event_time / last_event_time state, and disabling batching resumes
  unbatched queries from the latest safe watermark instead of replaying quiet
  windows that batching already drained.
  
  Tighten batching safety by rejecting object configs that enable batching
  without referencing both batch_start_time and batch_end_time, or that still
  reference those placeholders after batching is disabled.
  
  Prevent same-timestamp resume gaps for SetupAuditTrail and EventLogFile
  collection by persisting the last seen record Id as a tie-breaker. Existing
  installs remain compatible: older cursor state keeps the legacy resume boundary
  until the next successful run records the new field.
  
  Normalize Salesforce OAuth token_url handling for user-password and JWT flows.
  The input now accepts either a Salesforce OAuth host or the canonical
  /services/oauth2/token endpoint and avoids constructing doubled token URLs.
  
  Improve resilience by propagating input cancellation into Salesforce HTTP
  requests, retrying the initial SOQL request once after Salesforce auth errors,
  retrying EventLogFile downloads once after a 401, and surfacing consecutive
  collection failures in Elastic Agent status.
  
* Fix Okta minimal-state provider to accept configs with both okta_token and oauth2. [#51078](https://github.com/elastic/beats/pull/51078) [#51005](https://github.com/elastic/beats/issues/51005)
* Fix S3 polling input to exclude same-bucket backup objects from listing. [#51912](https://github.com/elastic/beats/pull/51912) 

  When same-bucket backup is configured with the default empty
  bucket_list_prefix, backup objects are listed alongside source objects
  and reprocessed indefinitely. Exclude keys matching the backup prefix
  from listing results when the backup destination is the same bucket.
  
* Honor path_style for non-AWS S3 buckets in the aws-s3 input. [#52003](https://github.com/elastic/beats/pull/52003) 

  When using non_aws_bucket_name with a custom endpoint, the aws-s3 input always marked the endpoint hostname immutable, forcing path-style requests and ignoring path_style. This broke S3-compatible providers that require virtual-hosted addressing, which returned VirtualHostDomainRequired.  The hostname is now kept mutable for non-AWS buckets so path_style is honored.
  
* Use a stable status message for SQS receive errors to prevent update churn during sustained outages. [#52059](https://github.com/elastic/beats/pull/52059) 
* Do not mark the CEL input&#39;s health as degraded when the maximum executions is exceeded. [#52060](https://github.com/elastic/beats/pull/52060) 
* Honor the configured language when rendering Windows events. [#52094](https://github.com/elastic/beats/pull/52094) [#7332](https://github.com/elastic/sdh-beats/issues/7332)
* Fix aws-s3 input silently dropping events when a parser clears the message content. [#52077](https://github.com/elastic/beats/pull/52077) 
* Fix journald input facility filters on journalctl older than 245. [#52231](https://github.com/elastic/beats/pull/52231) 
* The journald input now reports a Degraded status when journalctl repeatedly exits without delivering any data, instead of staying Healthy while collecting no events. [#52232](https://github.com/elastic/beats/pull/52232) 
* Fix TCP and UDP inputs hanging during shutdown. [#52292](https://github.com/elastic/beats/pull/52292) 

**Heartbeat**

* Fix panic with multiple heartbeat receivers. [#52010](https://github.com/elastic/beats/pull/52010) 

**Libbeat**

* Add LDAP channel binding for Windows SSPI binds. [#49733](https://github.com/elastic/beats/pull/49733) 
* Fix add_host_metadata host.geo map sharing across events causing data corruption. [#49722](https://github.com/elastic/beats/pull/49722) [#49721](https://github.com/elastic/beats/issues/49721)

**Osquerybeat**

* Reject CPIO entries in macOS .pkg payloads whose paths escape the destination directory, preventing path traversal during extraction (CWE-22). [#51446](https://github.com/elastic/beats/pull/51446) 
* Propagate osquery live-query space IDs to action response events. [#50808](https://github.com/elastic/beats/pull/50808) 
* Fixes serialization of ads and active columns in elastic_ntfs_file table to use boolean instead of integer. [#51629](https://github.com/elastic/beats/pull/51629) 
* Accept osquery packs mixing interval-scheduled and unscheduled queries again. [#52242](https://github.com/elastic/beats/pull/52242) [#51450](https://github.com/elastic/beats/issues/51450)

**Winlogbeat**

* Fix Kerberos ticket status code descriptions not being set in the Security event log ingest pipeline when `winlog.event_data.Status` arrives in lower case. [#51681](https://github.com/elastic/beats/pull/51681) 

