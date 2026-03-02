## 9.2.4 [beats-release-notes-9.2.4]



### Features and enhancements [beats-9.2.4-features-enhancements]


**Filebeat**

* Add client secret authentication method for Azure Event Hub and storage in Filebeat. [#47256](https://github.com/elastic/beats/pull/47256)
* Add support for AMQP-over-WebSocket transport in the processor v2. [#47956](https://github.com/elastic/beats/pull/47956) [#47823](https://github.com/elastic/beats/issues/47823)

**Metricbeat**

* Add `last_terminated_exitcode` to `kubernetes.container.status`. [#47968](https://github.com/elastic/beats/pull/47968)


### Fixes [beats-9.2.4-fixes]


**All**

* Upgrade opentelemtry-collector-contrib to 0.141.0 and opentelemetry-collector to 1.47.0. [#48122](https://github.com/elastic/beats/pull/48122)
* Add msync syscall to seccomp whitelist for BadgerDB persistent cache. [#48229](https://github.com/elastic/beats/pull/48229)

**Metricbeat**

* Harden Prometheus metrics parser against panics caused by malformed input data. [#47914](https://github.com/elastic/beats/pull/47914)
* Add bounds checking to Zookeeper server module to prevent index-out-of-range panics. [#47915](https://github.com/elastic/beats/pull/47915)
* Fix panic in graphite server metricset when metric has fewer parts than template expects. [#47916](https://github.com/elastic/beats/pull/47916)
* Skip regions with no permission to query for AWS CloudWatch metrics. [#48135](https://github.com/elastic/beats/pull/48135)

**Packetbeat**

* Fix bounds checking in MongoDB protocol parser to prevent panics. [#47925](https://github.com/elastic/beats/pull/47925)

