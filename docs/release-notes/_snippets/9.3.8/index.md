## 9.3.8 [beats-release-notes-9.3.8]



### Features and enhancements [beats-9.3.8-features-enhancements]


**All**

* Use native Go for Linux FIPS builds. [#51345](https://github.com/elastic/beats/pull/51345) 

**Filebeat**

* Optimize filestream logger performance. [#50118](https://github.com/elastic/beats/pull/50118) 
* Add consumer-group and network timeout options to the Filebeat Kafka input. [#51173](https://github.com/elastic/beats/pull/51173) [#51172](https://github.com/elastic/beats/issues/51172)
* Upgrade `github.com/elastic/go-lumber` to v0.2.0. [#51518](https://github.com/elastic/beats/pull/51518)
* Add a configurable retry policy to the Azure Blob Storage input. [#51701](https://github.com/elastic/beats/pull/51701) [#44629](https://github.com/elastic/beats/issues/44629)
* Add `group_instance_id` (KIP-345 static group membership) to the Filebeat Kafka input. [#51772](https://github.com/elastic/beats/pull/51772) [#51768](https://github.com/elastic/beats/issues/51768)

### Fixes [beats-9.3.8-fixes]


**All**

* Fix goroutine leak when processor construction fails. [#51687](https://github.com/elastic/beats/pull/51687) 
* Fix data races in the `add_docker_metadata` cache initialization. [#51688](https://github.com/elastic/beats/pull/51688) 
* Close Beat processors on Beat OTel processor shutdown to avoid leaking resources on collector reloads. [#51743](https://github.com/elastic/beats/pull/51743) 
* Update `elastic-agent-libs` to v0.46.1. [#51921](https://github.com/elastic/beats/pull/51921) 

**Auditbeat**

* Fix data races in the backoff accounting of the `add_session_metadata` processor's `kernel_tracing` provider. [#51745](https://github.com/elastic/beats/pull/51745) 

**Elastic agent**

* Fix registry corruption issue with multiple Metricbeat receivers. [#51591](https://github.com/elastic/beats/pull/51591) [#15154](https://github.com/elastic/elastic-agent/issues/15154)
  

**Filebeat**

* Filestream now defers shutdown until it reaches EOF or a configurable timeout. The new `read_until_eof` option (enabled by default) lets you opt out. [#50324](https://github.com/elastic/beats/pull/50324) [#40447](https://github.com/elastic/beats/issues/40447) 
* Fix DPoP resource client signing method assignment in CEL and HTTP JSON input. [#51433](https://github.com/elastic/beats/pull/51433) 
* Validate CrowdStrike streaming resource URL origins against the configured discover URL. [#51435](https://github.com/elastic/beats/pull/51435) 
* Use constant-time comparison for `http_endpoint` basic auth and secret header validation. [#51436](https://github.com/elastic/beats/pull/51436) 
* Validate that HTTPJSON pagination URLs share the configured request URL origin. [#51437](https://github.com/elastic/beats/pull/51437) 
* Fix Filebeat duplicating events after a normal shutdown caused by a race in the registrar. [#51517](https://github.com/elastic/beats/pull/51517) 
* Avoid slice-bounds panic when sorting copytruncate rotated files by date. [#51570](https://github.com/elastic/beats/pull/51570) 
* Fix type loss in HTTP JSON template transform handling. [#51593](https://github.com/elastic/beats/pull/51593) 
* Fix filestream data loss when a harvester closes before ingesting any data. [#51675](https://github.com/elastic/beats/pull/51675) 
* Fix CrowdStrike streaming input retry cap so max_attempts and infinite_retries are honoured. [#51712](https://github.com/elastic/beats/pull/51712) 
* Fix `winlog` input crash and event loss during shutdown. [#51728](https://github.com/elastic/beats/pull/51728) 
* Make the CrowdStrike streaming input self-heal from transient discover failures. [#51737](https://github.com/elastic/beats/pull/51737) 
* Prevent Filebeat startup failure when `meta.json` is left empty after migration. [#51791](https://github.com/elastic/beats/pull/51791) 
* Synchronize Filebeat Run and Shutdown functions. [#51800](https://github.com/elastic/beats/pull/51800) 
* Fix `aws-s3` input not performing `backup_to_bucket` and `delete_after_backup` in polling mode. [#49734](https://github.com/elastic/beats/pull/49734) [#46672](https://github.com/elastic/beats/issues/46672)

**Libbeat**

* Fix a data race in the `rate_limit` processor when configured with fields and run concurrently. [#51736](https://github.com/elastic/beats/pull/51736) 
* Fix a startup data race in the `add_kubernetes_metadata` processor. [#51739](https://github.com/elastic/beats/pull/51739) 
* Prevent statestore startup failure when `meta.json` is left empty after an unclean shutdown. [#51897](https://github.com/elastic/beats/pull/51897) 

**Metricbeat**

* Fix a race condition during metricset closure. [#50834](https://github.com/elastic/beats/pull/50834) 
* Fix missing `state_service` and `state_storageclass` events when `kube-state-metrics` denylists `*_created`. [#51255](https://github.com/elastic/beats/pull/51255) [#34074](https://github.com/elastic/beats/issues/34074)

**Osquerybeat**

* Fix a rare crash when Osquery restarts after a configuration change. [#51520](https://github.com/elastic/beats/pull/51520) 

**Packetbeat**

* Load the Npcap `wpcap.dll` lazily to avoid blocking Npcap upgrades on Windows. [#51716](https://github.com/elastic/beats/pull/51716) [#14517](https://github.com/elastic/elastic-agent/issues/14517)
