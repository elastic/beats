## 9.1.10 [beats-release-notes-9.1.10]



### Features and enhancements [beats-9.1.10-features-enhancements]


**Filebeat**

* Add client secret authentication method for Azure Event Hub and storage in Filebeat. [#48278](https://github.com/elastic/beats/pull/48278) [#48326](https://github.com/elastic/beats/pull/48326) [#48328](https://github.com/elastic/beats/pull/48328) [#48329](https://github.com/elastic/beats/pull/48329) [#47873](https://github.com/elastic/beats/pull/47873) [#48345](https://github.com/elastic/beats/pull/48345) [#48292](https://github.com/elastic/beats/pull/48292) 
* Improving input error reporting to Elastic Agent, specially when pipeline configurations are incorrect. [#47905](https://github.com/elastic/beats/pull/47905) [#45649](https://github.com/elastic/beats/issues/45649)
* Azure-eventhub-v2-add-support-for-amqp-over-websocket-and-https-proxy. [#47956](https://github.com/elastic/beats/pull/47956) [#47823](https://github.com/elastic/beats/issues/47823)

**Metricbeat**

* Add `last_terminated_exitcode` to `kubernetes.container.status`. [#48278](https://github.com/elastic/beats/pull/48278) [#48326](https://github.com/elastic/beats/pull/48326) [#48328](https://github.com/elastic/beats/pull/48328) [#48329](https://github.com/elastic/beats/pull/48329) [#47873](https://github.com/elastic/beats/pull/47873) [#48345](https://github.com/elastic/beats/pull/48345) [#48292](https://github.com/elastic/beats/pull/48292) 


### Fixes [beats-9.1.10-fixes]


**All**

* Add msync syscall to seccomp whitelist for BadgerDB persistent cache. [#48278](https://github.com/elastic/beats/pull/48278) [#48326](https://github.com/elastic/beats/pull/48326) [#48328](https://github.com/elastic/beats/pull/48328) [#48329](https://github.com/elastic/beats/pull/48329) [#47873](https://github.com/elastic/beats/pull/47873) [#48345](https://github.com/elastic/beats/pull/48345) [#48292](https://github.com/elastic/beats/pull/48292) 

**Metricbeat**

* Harden Prometheus metrics parser against panics caused by malformed input data. [#48278](https://github.com/elastic/beats/pull/48278) [#48326](https://github.com/elastic/beats/pull/48326) [#48328](https://github.com/elastic/beats/pull/48328) [#48329](https://github.com/elastic/beats/pull/48329) [#47873](https://github.com/elastic/beats/pull/47873) [#48345](https://github.com/elastic/beats/pull/48345) [#48292](https://github.com/elastic/beats/pull/48292) 
* Add bounds checking to Zookeeper server module to prevent index-out-of-range panics. [#48278](https://github.com/elastic/beats/pull/48278) [#48326](https://github.com/elastic/beats/pull/48326) [#48328](https://github.com/elastic/beats/pull/48328) [#48329](https://github.com/elastic/beats/pull/48329) [#47873](https://github.com/elastic/beats/pull/47873) [#48345](https://github.com/elastic/beats/pull/48345) [#48292](https://github.com/elastic/beats/pull/48292) 
* Fix panic in graphite server metricset when metric has fewer parts than template expects. [#48278](https://github.com/elastic/beats/pull/48278) [#48326](https://github.com/elastic/beats/pull/48326) [#48328](https://github.com/elastic/beats/pull/48328) [#48329](https://github.com/elastic/beats/pull/48329) [#47873](https://github.com/elastic/beats/pull/47873) [#48345](https://github.com/elastic/beats/pull/48345) [#48292](https://github.com/elastic/beats/pull/48292) 
* Skip regions with no permission to query for AWS CloudWatch metrics. [#48278](https://github.com/elastic/beats/pull/48278) [#48326](https://github.com/elastic/beats/pull/48326) [#48328](https://github.com/elastic/beats/pull/48328) [#48329](https://github.com/elastic/beats/pull/48329) [#47873](https://github.com/elastic/beats/pull/47873) [#48345](https://github.com/elastic/beats/pull/48345) [#48292](https://github.com/elastic/beats/pull/48292) 

**Packetbeat**

* Fix bounds checking in MongoDB protocol parser to prevent panics. [#48278](https://github.com/elastic/beats/pull/48278) [#48326](https://github.com/elastic/beats/pull/48326) [#48328](https://github.com/elastic/beats/pull/48328) [#48329](https://github.com/elastic/beats/pull/48329) [#47873](https://github.com/elastic/beats/pull/47873) [#48345](https://github.com/elastic/beats/pull/48345) [#48292](https://github.com/elastic/beats/pull/48292) 

