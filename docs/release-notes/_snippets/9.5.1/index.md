## 9.5.1 [beats-release-notes-9.5.1]





### Fixes [beats-9.5.1-fixes]


**All**

* Fix unknown type for `map[string]string` in `otelmap`. [#52464](https://github.com/elastic/beats/pull/52464) 

**Filebeat**

* Fix offset commits when using multiline in Journald and Kafka inputs. [#52342](https://github.com/elastic/beats/pull/52342) [#51981](https://github.com/elastic/beats/issues/51981)

**Heartbeat**

* Bake Synthetics browser binaries into the Heartbeat Docker image. [#52443](https://github.com/elastic/beats/pull/52443) [#52439](https://github.com/elastic/beats/issues/52439)

**Libbeat**

* Prevent stale autodiscover resources after Kubernetes leadership changes. [#52425](https://github.com/elastic/beats/pull/52425) 

**Metricbeat**

* Fix Azure module authentication on sovereign clouds (Government, China). [#52432](https://github.com/elastic/beats/pull/52432) 

