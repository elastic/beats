## 9.5.5 [beats-release-notes-9.5.5]



### Features and enhancements [beats-9.5.5-features-enhancements]


**All**

* Update Go to 1.26.8. [#53026](https://github.com/elastic/beats/pull/53026) 

**Filebeat**

* Add `api_endpoint` configuration option to `gcp-pubsub` input to allow overriding the default Pub/Sub API endpoint. [#53080](https://github.com/elastic/beats/pull/53080) [#48923](https://github.com/elastic/beats/issues/48923)
* Filestream `take_over` supports `from_any_id` to migrate from any previous filestream input. [#53122](https://github.com/elastic/beats/pull/53122) [#51307](https://github.com/elastic/beats/issues/51307)
* Reuse the UDP and Unix datagram read buffer so each packet allocates only its own length. [#53490](https://github.com/elastic/beats/pull/53490) 

**Libbeat**

* Add `append_fields` option to `add_kubernetes_metadata` processor to merge metadata into events that already have a `kubernetes` field without overwriting existing keys. [#53098](https://github.com/elastic/beats/pull/53098) [#53094](https://github.com/elastic/beats/issues/53094)
* Reduce per-event allocations in the `add_agent_metadata` processor by pre-building the static metadata maps once at construction. [#53477](https://github.com/elastic/beats/pull/53477) 

**Metricbeat**

* Persist additional cluster settings in the `autoops_es` `cluster_settings` metricset. [#53145](https://github.com/elastic/beats/pull/53145) 
* Report archived cluster setting names in the `autoops_es` `cluster_settings` metricset. [#53146](https://github.com/elastic/beats/pull/53146) [#52654](https://github.com/elastic/beats/issues/52654)


### Fixes [beats-9.5.5-fixes]


**Auditbeat**

* Fix Auditbeat `file_integrity` kprobe leak that could prevent FIM from starting when the running kernel does not match the first candidate BTF spec. [#53253](https://github.com/elastic/beats/pull/53253) [#45827](https://github.com/elastic/beats/issues/45827)
* Fix process misattribution in `system/socket` where TCP/UDP flows were attributed to wrong processes. [#53421](https://github.com/elastic/beats/pull/53421) [#44741](https://github.com/elastic/beats/issues/44741)

**Filebeat**

* Fix the Elasticsearch state store writing other inputs of the same type to the first input's index. [#53178](https://github.com/elastic/beats/pull/53178) [#53034](https://github.com/elastic/beats/issues/53034)
* Reconnect the ETW input when its real-time session is stopped and restarted. [#53112](https://github.com/elastic/beats/pull/53112) [#49984](https://github.com/elastic/beats/issues/49984)
* Fix race condition in Kafka input when multiple inputs are configured. [#53115](https://github.com/elastic/beats/pull/53115) [#34919](https://github.com/elastic/beats/issues/34919)
* Fix Azure AD entity analytics accumulating stale registered owners and users on devices. [#53134](https://github.com/elastic/beats/pull/53134) [#53132](https://github.com/elastic/beats/issues/53132)
* Fix Active Directory entity analytics panic when `ssl.certificate_authorities` is set. [#53218](https://github.com/elastic/beats/pull/53218) 
* Fix EntraID entity analytics collection behavior for deleted devices when using minimal state mode. [#53254](https://github.com/elastic/beats/pull/53254) [#20](https://github.com/elastic/entcollect/issues/20)
* Respond to `http_endpoint` requests that contain no events instead of closing the connection. [#53261](https://github.com/elastic/beats/pull/53261)
* Fix `httpjson` rate limiter marking the input "Degraded" when rate-limit headers are absent. [#53326](https://github.com/elastic/beats/pull/53326) [#53185](https://github.com/elastic/beats/issues/53185)
* Scope the `aws-s3` lexicographical polling tail to the input's bucket and key prefix. [#53351](https://github.com/elastic/beats/pull/53351) [#53195](https://github.com/elastic/beats/issues/53195)

**Libbeat**

* Fix `add_kubernetes_metadata`: `wait_for_metadata` now prevents unenriched events at startup. [#53093](https://github.com/elastic/beats/pull/53093) [#53090](https://github.com/elastic/beats/issues/53090)
* Make an ETW reader session safe to reuse and stop, and fix a lost stop that could hang shutdown. [#53112](https://github.com/elastic/beats/pull/53112) [#49984](https://github.com/elastic/beats/issues/49984)
* Fix healthy input units stuck in `CONFIGURING` when a sibling unit fails config validation. [#53299](https://github.com/elastic/beats/pull/53299) [#53292](https://github.com/elastic/beats/issues/53292)

**Metricbeat**

* Stop losing a day of `azure/billing` data when Cost Management returns HTTP 429. [#53129](https://github.com/elastic/beats/pull/53129)

**Osquerybeat**

* Bound the `osqueryd` startup check and report startup failures through Elastic Agent. [#52992](https://github.com/elastic/beats/pull/52992)
* Clamp native schedule metadata and history timestamps when endpoint clock skew precedes the configured start date. [#52992](https://github.com/elastic/beats/pull/52992) 

**Packetbeat**

* Fix rendering of TLS certificate serial numbers so each byte is always represented as two hex digits, matching OpenSSL output. [#53027](https://github.com/elastic/beats/pull/53027) [#52995](https://github.com/elastic/beats/issues/52995)
* Fix panic in DHCPv4 parser when the `OptionTimeOffset` value is shorter than 4 bytes. [#53419](https://github.com/elastic/beats/pull/53419) [#53330](https://github.com/elastic/beats/issues/53330)

**Winlogbeat**

* Fix binary event data fields being overread into subsequent buffer contents. [#53327](https://github.com/elastic/beats/pull/53327) [#53260](https://github.com/elastic/beats/issues/53260)

