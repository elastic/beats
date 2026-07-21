## 9.3.8 [beats-release-notes-9.3.8]



### Features and enhancements [beats-9.3.8-features-enhancements]


**All**

* Use native Go for Linux FIPS builds. [#51345](https://github.com/elastic/beats/pull/51345) 

**Filebeat**

* Optimize filestream logger performance. [#50118](https://github.com/elastic/beats/pull/50118) 
* Add consumer-group and network timeout options to the Filebeat Kafka input. [#51173](https://github.com/elastic/beats/pull/51173) [#51172](https://github.com/elastic/beats/issues/51172)

  The Filebeat Kafka input now exposes the `session_timeout`, `heartbeat_interval`, `timeout` and `keep_alive` options, which are passed through to the underlying Sarama consumer (`Consumer.Group.Session.Timeout`, `Consumer.Group.Heartbeat.Interval` and the `Net` dial/read/write timeouts). Previously these were pinned to Sarama&#39;s defaults with no way to override them, which prevented tuning consumers that read across higher-latency (cross-region/WAN) links. Defaults are unchanged.
  
* Upgrade github.com/elastic/go-lumber to v0.2.0. [#51518](https://github.com/elastic/beats/pull/51518) 
* Add a configurable retry policy to the Azure Blob Storage input. [#51701](https://github.com/elastic/beats/pull/51701) [#44629](https://github.com/elastic/beats/issues/44629)

  The `azure-blob-storage` input now exposes a `retry` configuration block (`max_retries`, `initial_retry_delay` and `max_retry_delay`) that tunes the Azure SDK retry policy. Because the policy lives in the client request pipeline, it now covers blob listing (pagination) in addition to downloads; previously a transient error such as an HTTP 503 `ServerBusy` during listing could exhaust the small, non-configurable default retries and stop the input. The settings apply per storage account. Defaults are unchanged when the block is omitted. Additionally, when polling is enabled, a transient blob-listing failure that outlives the retries (503, 429 or a network timeout) is now non-fatal: the input is marked degraded and retries on the next poll interval instead of exiting, so longer outages are ridden out. Permanent failures such as a missing container still stop the input.
  
* Add `group_instance_id` (KIP-345 static group membership) to the Filebeat Kafka input. [#51772](https://github.com/elastic/beats/pull/51772) [#51768](https://github.com/elastic/beats/issues/51768)

  The Filebeat Kafka input now exposes `group_instance_id`, wired to Sarama&#39;s `Consumer.Group.InstanceId`. Setting a stable, unique id per consumer instance enables Kafka static group membership (KIP-345): a member that restarts and rejoins within `session_timeout` is recognized as the same member, avoiding the rebalance storms that dynamic membership causes during rolling restarts of multi-replica deployments. Requires `version` &gt;= 2.3.0. Unset by default, so existing behavior is unchanged.
  


### Fixes [beats-9.3.8-fixes]


**All**

* Fix goroutine leak when processor construction fails. [#51687](https://github.com/elastic/beats/pull/51687) 
* Fix data races in the add_docker_metadata cache initialization. [#51688](https://github.com/elastic/beats/pull/51688) 
* Close Beat processors on beat OTel processor shutdown to avoid leaking resources on collector reloads. [#51743](https://github.com/elastic/beats/pull/51743) 
* Update elastic-agent-libs to v0.46.1. [#51921](https://github.com/elastic/beats/pull/51921) 

  Fixed an issue where malformed TLS keys could be printed in the error logs during loading failures.

**Auditbeat**

* Fix data races in the add_session_metadata kernel_tracing provider backoff accounting. [#51745](https://github.com/elastic/beats/pull/51745) 

**Elastic agent**

* Fix registry corruption issue with multiple metricbeat receivers. [#51591](https://github.com/elastic/beats/pull/51591) [#15154](https://github.com/elastic/elastic-agent/issues/15154)

  Change paths definition from global paths to per beat instance paths.  Without this multiple metricbeat receivers could use the same data store.
  

**Filebeat**

* Filestream now defers shutdown until it reaches EOF or a configurable timeout. The new read_until_eof option (enabled by default) lets users opt out. [#50324](https://github.com/elastic/beats/pull/50324) [#40447](https://github.com/elastic/beats/issues/40447)
* Fix DPoP resource client signing method assignment in CEL and HTTP JSON input. [#51433](https://github.com/elastic/beats/pull/51433) 
* Validate CrowdStrike streaming resource URL origins against the configured discover URL. [#51435](https://github.com/elastic/beats/pull/51435) 
* Use constant-time comparison for http_endpoint basic auth and secret header validation. [#51436](https://github.com/elastic/beats/pull/51436) 
* Validate that HTTPJSON pagination URLs share the configured request URL origin. [#51437](https://github.com/elastic/beats/pull/51437) 
* Fix Filebeat duplicating events after a normal shutdown caused by a race in the registrar. [#51517](https://github.com/elastic/beats/pull/51517) 
* Avoid slice-bounds panic when sorting copytruncate rotated files by date. [#51570](https://github.com/elastic/beats/pull/51570) 

  When the filestream copytruncate prospector sorts rotated files using a date format, `dateSorter.GetTs` sliced the file path by the format length without checking the path was long enough. A path shorter than the configured date format triggered a &#34;slice bounds out of range&#34; panic inside a prospector goroutine, crashing the process. GetTs now returns a zero time for paths shorter than the format, mirroring the existing parse-failure path.
  
* Fix type loss in HTTP JSON template transform handling. [#51593](https://github.com/elastic/beats/pull/51593) 
* Fix filestream data loss when a harvester closes before ingesting any data. [#51675](https://github.com/elastic/beats/pull/51675) 

  When a filestream harvester was closed before ingesting anything (for example while still in its initial backoff), it reported an ingested offset of 0. The file watcher could not distinguish this genuine 0 from &#34;no offset reported&#34;, so it fell back to the file size and never emitted a write event. As a result no new harvester was started and the file&#39;s contents were never ingested, causing silent data loss. The reported offset of 0 is now honored, so a new harvester is started and the file is ingested.
  
* Fix CrowdStrike streaming input retry cap so max_attempts and infinite_retries are honoured. [#51712](https://github.com/elastic/beats/pull/51712) 

  Fix the CrowdStrike streaming input retry loop so a configured max_attempts
  greater than 10 and infinite_retries are no longer silently capped at the
  unconfigured default of 10. The empty-body case from the discover endpoint is
  now reported as a distinct transient error, and the input no longer reports
  DEGRADED on the first transient failure.
  
* Fix winlog input crash and event loss during shutdown. [#51728](https://github.com/elastic/beats/pull/51728) 

  The winlog input could crash with a native access violation (0xc0000005) during shutdown when the event log was closed while a Read was still rendering an event. The event log was closed asynchronously from a context-cancellation callback on a different goroutine than the read loop, so closing freed the native Windows Event Log handles (subscription, render contexts and publisher metadata) while in-flight EvtRender/EvtFormatMessage calls were still using them. The crash restarted the component and dropped in-flight events. The event log is now closed on the same goroutine as the read loop, after it has stopped, eliminating the race.
  
* Make the CrowdStrike streaming input self-heal from transient discover failures. [#51737](https://github.com/elastic/beats/pull/51737) 

  Transient connection-level failures from the CrowdStrike discover endpoint
  (an empty 200 body, network errors and timeouts) are now retried indefinitely
  with capped back-off instead of counting toward max_attempts and terminating
  the input, so the input self-heals once the upstream recovers rather than
  needing an agent restart. Termination is reserved for genuine hard errors
  (origin mismatch, publish failure); other soft errors, including OAuth auth
  failures from bad credentials, still honour the configured attempt limit.
  
* Prevent Filebeat startup failure when meta.json is left empty after migration. [#51791](https://github.com/elastic/beats/pull/51791) 
* Synchronize Filebeat Run and Shutdown functions. [#51800](https://github.com/elastic/beats/pull/51800) 
* Fix aws-s3 input not performing backup_to_bucket and delete_after_backup in polling mode. [#49734](https://github.com/elastic/beats/pull/49734) [#46672](https://github.com/elastic/beats/issues/46672)

**Libbeat**

* Fix a data race in the rate_limit processor when configured with fields and run concurrently. [#51736](https://github.com/elastic/beats/pull/51736) 
* Fix a startup data race in the add_kubernetes_metadata processor. [#51739](https://github.com/elastic/beats/pull/51739) 
* Prevent statestore startup failure when meta.json is left empty after an unclean shutdown. [#51897](https://github.com/elastic/beats/pull/51897) 

**Metricbeat**

* Fix race condition during metricset closure. [#50834](https://github.com/elastic/beats/pull/50834) 
* Fix missing state_service and state_storageclass events when kube-state-metrics denylists *_created. [#51255](https://github.com/elastic/beats/pull/51255) [#34074](https://github.com/elastic/beats/issues/34074)

**Osquerybeat**

* Fix a rare crash when osquery restarts after a configuration change. [#51520](https://github.com/elastic/beats/pull/51520) 

**Packetbeat**

* Load the Npcap wpcap.dll lazily to avoid blocking Npcap upgrades on Windows. [#51716](https://github.com/elastic/beats/pull/51716) [#14517](https://github.com/elastic/elastic-agent/issues/14517)

  On Windows, importing gopacket/pcap loaded wpcap.dll in the package init, so every Beat that links Packetbeat&#39;s capture code held the DLL open even when it never captured traffic. This stopped Packetbeat from replacing wpcap.dll while upgrading Npcap and the install failed. The DLL is now loaded lazily, only when Packetbeat needs it.
  

