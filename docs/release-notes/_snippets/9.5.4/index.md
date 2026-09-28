## 9.5.4 [beats-release-notes-9.5.4]



### Features and enhancements [beats-9.5.4-features-enhancements]


**All**

* Add hostname overrides for Beat events. [#53029](https://github.com/elastic/beats/pull/53029) [#48923](https://github.com/elastic/beats/issues/48923)

**Filebeat**

* Add `hints.input_allow_list` to restrict Filebeat hints-generated input types. [#53006](https://github.com/elastic/beats/pull/53006) [#52624](https://github.com/elastic/beats/issues/52624)
* Report health status from the ETW input when running under Elastic Agent. [#53089](https://github.com/elastic/beats/pull/53089) [#44271](https://github.com/elastic/beats/issues/44271)
  

**Metricbeat**

* Add ingest volume enrichment fields to the `autoops_es` module. [#52914](https://github.com/elastic/beats/pull/52914) [#48923](https://github.com/elastic/beats/issues/48923)
* Collect `logical_availability_zone` and `instance_configuration` node attributes in the `autoops_es` `node_stats` metricset. [#53119](https://github.com/elastic/beats/pull/53119) [#48923](https://github.com/elastic/beats/issues/48923)
* Collect the frozen flood stage disk watermark in AutoOps cluster settings. [#53114](https://github.com/elastic/beats/pull/53114) [#52341](https://github.com/elastic/beats/issues/52341)


### Fixes [beats-9.5.4-fixes]


**Filebeat**

* Wire Sarama client logs in the Filebeat Kafka input so consumer-group lifecycle is visible. [#52980](https://github.com/elastic/beats/pull/52980) [#51305](https://github.com/elastic/beats/issues/51305)
* Prevent `httpjson` chain pagination from advancing the cursor when a chained request fails. [#53063](https://github.com/elastic/beats/pull/53063) [#48923](https://github.com/elastic/beats/issues/48923)
* Follow every CrowdStrike discover resource instead of only the first feed. [#53060](https://github.com/elastic/beats/pull/53060) [#48923](https://github.com/elastic/beats/issues/48923)
* Update `github.com/elastic/entcollect` to fix bugs in the EntraID and Okta providers. [#52971](https://github.com/elastic/beats/pull/52971) [#48923](https://github.com/elastic/beats/issues/48923)

**Heartbeat**

* Align maintenance window recurrence handling with Kibana, including nth-weekday rules, `tzid` timezone evaluation, `until` end bounds, hourly recurrences, inclusive duration behavior, and legacy `dtstart` formats. [#52957](https://github.com/elastic/beats/pull/52957) [#52571](https://github.com/elastic/beats/issues/52571)

**Metricbeat**

* Fix Kubernetes metadata watcher deadlock that blocked metricset shutdown. [#53040](https://github.com/elastic/beats/pull/53040) [#48923](https://github.com/elastic/beats/issues/48923)
  

**Osquerybeat**

* Restart `osqueryd` instead of terminating Osquerybeat when the extension manager ping times out. [#52993](https://github.com/elastic/beats/pull/52993) [#48923](https://github.com/elastic/beats/issues/48923)
  

