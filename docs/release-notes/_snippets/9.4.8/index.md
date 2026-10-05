## 9.4.8 [beats-release-notes-9.4.8]



### Features and enhancements [beats-9.4.8-features-enhancements]


**All**

* Add hostname overrides for Beat events. [#52126](https://github.com/elastic/beats/pull/52126)
* Update Go to 1.26.8. [#53026](https://github.com/elastic/beats/pull/53026) 

**Filebeat**

* Improve `add_kubernetes_metadata`: compressed rotated log files (for example, `0.log.gz`) matched via `/var/log/pods/` with `resource_type:pod` are now enriched with pod and container metadata. Previously they were excluded from enrichment entirely. [#53300](https://github.com/elastic/beats/pull/53300) [#48922](https://github.com/elastic/beats/issues/48922)
* Add `api_endpoint` configuration option to `gcp-pubsub` input to allow overriding the default Pub/Sub API endpoint. [#53080](https://github.com/elastic/beats/pull/53080) 
* Filestream `take_over` supports `from_any_id` to migrate from any previous filestream input.  [#51307](https://github.com/elastic/beats/issues/51307)
* Reuse the UDP and Unix datagram read buffer so each packet allocates only its own length.

**Libbeat**

* Improve `add_kubernetes_metadata`: events matched from `/var/log/pods/` with `resource_type:pod` now include `kubernetes.container.name` and `container.image.name` (always), plus `container.id` and `container.runtime` for the currently-running log file. The immediately preceding restart also receives `container.id` and `container.runtime` when its container ID is available from `LastTerminationState`. Older rotated log files receive `kubernetes.container.name` and `container.image.name` only. [#53300](https://github.com/elastic/beats/pull/53300) [#48922](https://github.com/elastic/beats/issues/48922)
* Add `append_fields` option to `add_kubernetes_metadata` processor to merge metadata into events that already have a `kubernetes` field without overwriting existing keys.  
* Reduce per-event allocations in the `add_agent_metadata` processor by pre-building the static metadata maps once at construction.  

**Metricbeat**

* Persist additional cluster settings in the `autoops_es` `cluster_settings` metricset.  
* Report archived cluster setting names in the `autoops_es` `cluster_settings` metricset.  


### Fixes [beats-9.4.8-fixes]


**Auditbeat**

* Fix auditbeat file_integrity kprobe leak that could prevent FIM from starting when the running kernel does not match the first candidate BTF spec.  
* Fix process misattribution in system/socket where TCP/UDP flows were attributed to wrong processes.  

**Filebeat**

* Fix `add_kubernetes_metadata`: the `logs_path` matcher now requires `logs_path` to be a true prefix of the log file path (`HasPrefix`). Previously it used a substring check (`Contains`), which would accept paths like `/mnt/host/var/log/pods/...` and then index into the wrong path segments. Users relying on the old behavior should update their `logs_path` to match the full path prefix of their log files. [#53300](https://github.com/elastic/beats/pull/53300) [#48922](https://github.com/elastic/beats/issues/48922)
* Wire Sarama client logs in the Filebeat Kafka input so consumer-group lifecycle is visible. [#52871](https://github.com/elastic/beats/pull/52871) [#51305](https://github.com/elastic/beats/issues/51305)
* Reconnect the ETW input when its real-time session is stopped and restarted. [#49984](https://github.com/elastic/beats/issues/49984)
* Fix race condition in Kafka input when multiple inputs are configured.  [#34919](https://github.com/elastic/beats/issues/34919)
* Fix Azure AD entity analytics accumulating stale registered owners and users on devices.
* Fix Active Directory entity analytics panic when ssl.certificate_authorities is set.  
* Respond to `http_endpoint` requests that contain no events instead of closing the connection.
* Fix `httpjson` rate limiter marking the input "Degraded" when rate-limit headers are absent.
* Scope the `aws-s3` lexicographical polling tail to the input's bucket and key prefix. [#53195](https://github.com/elastic/beats/issues/53195)

**Libbeat**

* Fix `add_kubernetes_metadata`: `wait_for_metadata` now prevents unenriched events at startup. [#53093](https://github.com/elastic/beats/pull/53093) [#53090](https://github.com/elastic/beats/issues/53090)
* Fix memory leak in `add_kubernetes_metadata` where old restart-count cache entries (for example, `uid/container/N`) were never scheduled for eviction when a pod's restart count changed. The annotator now records all keys written per pod and deletes exactly that set on the next update. [#53300](https://github.com/elastic/beats/pull/53300) [#48922](https://github.com/elastic/beats/issues/48922)
* Make an ETW reader session safe to reuse and stop, and fix a lost stop that could hang shutdown. [#49984](https://github.com/elastic/beats/issues/49984)
* Fix healthy input units stuck in `CONFIGURING` when a sibling unit fails config validation. [#53292](https://github.com/elastic/beats/issues/53292)

**Metricbeat**

* Stop losing a day of `azure/billing` data when Cost Management returns HTTP 429.

**Osquerybeat**

* Bound the `osqueryd` startup check and report startup failures through Elastic Agent.
* Clamp native schedule metadata and history timestamps when endpoint clock skew precedes the configured start date.  

**Packetbeat**

* Reject RESP bulk-string lengths other than `-1` (nil) and `>=0` in the Packetbeat Redis parser. [#50288](https://github.com/elastic/beats/pull/50288) [#50205](https://github.com/elastic/beats/issues/50205)
* Fix rendering of TLS certificate serial numbers so each byte is always represented as two hex digits, matching OpenSSL output.  
* Fix panic in DHCPv4 parser when the `OptionTimeOffset` value is shorter than 4 bytes.

**Winlogbeat**

* Fix binary event data fields being overread into subsequent buffer contents.  

