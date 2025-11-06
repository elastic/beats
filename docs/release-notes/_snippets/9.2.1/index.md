## 9.2.1 [beats-release-notes-9.2.1]



### Features and enhancements [beats-9.2.1-features-enhancements]


**Filebeat**

* Add data stream identification to Fleet health status updates. [#47229](https://github.com/elastic/beats/pull/47229) 

**Metricbeat**

* Enhance GCP Billing metricset with additional fields. [#47059](https://github.com/elastic/beats/pull/47059) 


### Fixes [beats-9.2.1-fixes]


**All**

* Add close to conditional processors if underlying processors have close method. [#46653](https://github.com/elastic/beats/pull/46653) [#46575](https://github.com/elastic/beats/issues/46575)
* Fixes a bug where kerberos authentication could be disabled when server supports multiple authentication types. [#47444](https://github.com/elastic/beats/pull/47444) [#47443](https://github.com/elastic/beats/pull/47443) [#47110](https://github.com/elastic/beats/issues/47110)

**Filebeat**

* Fix potential Filebeat panic during memory queue shutdown. [#47248](https://github.com/elastic/beats/pull/47248) 

