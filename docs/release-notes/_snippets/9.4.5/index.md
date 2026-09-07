## 9.4.5 [beats-release-notes-9.4.5]



### Features and enhancements [beats-9.4.5-features-enhancements]


**Auditbeat**

* Update quark to 0.6.0. [#51767](https://github.com/elastic/beats/pull/51767) 

**Filebeat**

* Aggregate SQS mode health status to reduce false Degraded/Healthy transitions. [#51819](https://github.com/elastic/beats/pull/51819) [#51692](https://github.com/elastic/beats/issues/51692)
* Add configurable User-Agent header to the CrowdStrike streaming input. [#51822](https://github.com/elastic/beats/pull/51822) 
* Remove Authorization headers from CEL, Entity Analytics, HTTP Endpoint, and HTTP JSON input request trace logs.  


### Fixes [beats-9.4.5-fixes]


**All**

* Upgrade Go to 1.26.5.  

**Elastic Agent**

* Fix Beat receiver Shutdown hanging when Start fails before the beater is launched. [#52106](https://github.com/elastic/beats/pull/52106) [#52009](https://github.com/elastic/beats/issues/52009)

**Filebeat**

* Fix offset commits when using multiline in Journald and Kafka inputs. [#52286](https://github.com/elastic/beats/pull/52286) [#51981](https://github.com/elastic/beats/issues/51981)
* Fix S3 polling input to exclude same-bucket backup objects from listing. [#52170](https://github.com/elastic/beats/pull/52170)
* Honor `path_style` for non-AWS S3 buckets in the `aws-s3` input. [#52003](https://github.com/elastic/beats/pull/52003)
* Use a stable status message for SQS receive errors to prevent update churn during sustained outages.  
* Do not mark the CEL input's health as degraded when the maximum number of executions is exceeded.  
* Fix the `aws-s3` silently dropping events when a parser clears the message content. [#52077](https://github.com/elastic/beats/pull/52077)
* Honor the configured language when rendering Windows events. [#7332](https://github.com/elastic/sdh-beats/issues/7332)
* Fix `journald` input facility filters on `journalctl` older than 245.
* The `journald` input now reports a Degraded status when `journalctl` repeatedly exits without delivering any data, instead of staying Healthy while collecting no events.  
* Fix TCP and UDP inputs hanging during shutdown.  

**Heartbeat**

* Fix panic with multiple Heartbeat receivers.  
* Bake Synthetics browser binaries into the Heartbeat Docker image.  [#52439](https://github.com/elastic/beats/issues/52439)

**Libbeat**

* Prevent stale autodiscover resources after Kubernetes leadership changes.  

**Metricbeat**

* Fix Azure module authentication on sovereign clouds (Government, China).

**Osquerybeat**

* Reject CPIO entries in macOS .pkg payloads whose paths escape the destination directory, preventing path traversal during extraction (CWE-22). [#51446](https://github.com/elastic/beats/pull/51446) 

