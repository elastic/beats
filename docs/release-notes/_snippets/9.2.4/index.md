## 9.2.4 [beats-release-notes-9.2.4]



### Features and enhancements [beats-9.2.4-features-enhancements]


**Filebeat**

* Add client secret authentication method for Azure Event Hub and storage in Filebeat. [#48290](https://github.com/elastic/beats/pull/48290) [#48330](https://github.com/elastic/beats/pull/48330) [#48331](https://github.com/elastic/beats/pull/48331) [#48332](https://github.com/elastic/beats/pull/48332) [#48292](https://github.com/elastic/beats/pull/48292) 
* Azure-eventhub-v2-add-support-for-amqp-over-websocket-and-https-proxy. [#47956](https://github.com/elastic/beats/pull/47956) [#47823](https://github.com/elastic/beats/issues/47823)

**Metricbeat**

* Add `last_terminated_exitcode` to `kubernetes.container.status`. [#48290](https://github.com/elastic/beats/pull/48290) [#48330](https://github.com/elastic/beats/pull/48330) [#48331](https://github.com/elastic/beats/pull/48331) [#48332](https://github.com/elastic/beats/pull/48332) [#48292](https://github.com/elastic/beats/pull/48292) 


### Fixes [beats-9.2.4-fixes]


**All**

* Add msync syscall to seccomp whitelist for BadgerDB persistent cache. [#48290](https://github.com/elastic/beats/pull/48290) [#48330](https://github.com/elastic/beats/pull/48330) [#48331](https://github.com/elastic/beats/pull/48331) [#48332](https://github.com/elastic/beats/pull/48332) [#48292](https://github.com/elastic/beats/pull/48292) 

**Metricbeat**

* Harden Prometheus metrics parser against panics caused by malformed input data. [#48290](https://github.com/elastic/beats/pull/48290) [#48330](https://github.com/elastic/beats/pull/48330) [#48331](https://github.com/elastic/beats/pull/48331) [#48332](https://github.com/elastic/beats/pull/48332) [#48292](https://github.com/elastic/beats/pull/48292) 
* Add bounds checking to Zookeeper server module to prevent index-out-of-range panics. [#48290](https://github.com/elastic/beats/pull/48290) [#48330](https://github.com/elastic/beats/pull/48330) [#48331](https://github.com/elastic/beats/pull/48331) [#48332](https://github.com/elastic/beats/pull/48332) [#48292](https://github.com/elastic/beats/pull/48292) 
* Fix panic in graphite server metricset when metric has fewer parts than template expects. [#48290](https://github.com/elastic/beats/pull/48290) [#48330](https://github.com/elastic/beats/pull/48330) [#48331](https://github.com/elastic/beats/pull/48331) [#48332](https://github.com/elastic/beats/pull/48332) [#48292](https://github.com/elastic/beats/pull/48292) 
* Skip regions with no permission to query for AWS CloudWatch metrics. [#48290](https://github.com/elastic/beats/pull/48290) [#48330](https://github.com/elastic/beats/pull/48330) [#48331](https://github.com/elastic/beats/pull/48331) [#48332](https://github.com/elastic/beats/pull/48332) [#48292](https://github.com/elastic/beats/pull/48292) 

**Packetbeat**

* Fix bounds checking in MongoDB protocol parser to prevent panics. [#48290](https://github.com/elastic/beats/pull/48290) [#48330](https://github.com/elastic/beats/pull/48330) [#48331](https://github.com/elastic/beats/pull/48331) [#48332](https://github.com/elastic/beats/pull/48332) [#48292](https://github.com/elastic/beats/pull/48292) 

