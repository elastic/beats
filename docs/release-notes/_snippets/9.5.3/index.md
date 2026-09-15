## 9.5.3 [beats-release-notes-9.5.3]



### Features and enhancements [beats-9.5.3-features-enhancements]


**All**

* Update Go to 1.26.7. [#52750](https://github.com/elastic/beats/pull/52750) 

**Filebeat**

* Reduce filestream memory usage with multiple Filebeat receivers. [#52326](https://github.com/elastic/beats/pull/52326)
  

**Packetbeat**

* Log Npcap installer diagnostic files on installation failure. [#52667](https://github.com/elastic/beats/pull/52667) 


### Fixes [beats-9.5.3-fixes]


**Filebeat**

* Fix the `auditd` filestream parser to emit `unset` for unset `auid` and `ses` fields. [#52614](https://github.com/elastic/beats/pull/52614) 
* Fix CEL input panic on nil rate-limit fields and 429 retry hot-loop. [#52684](https://github.com/elastic/beats/pull/52684) 
* Scope the `aws-s3` polling state registry to the input's bucket. [#52728](https://github.com/elastic/beats/pull/52728) [#52721](https://github.com/elastic/beats/issues/52721)  
* Log entity analytics sync interruption by input shutdown or reconfiguration at info level instead of error, and no longer mark the input as degraded. [#52950](https://github.com/elastic/beats/pull/52950) [#52945](https://github.com/elastic/beats/issues/52945)

**Osquerybeat**

* Emit `pack_name` and `query_name` in Osquerybeat scheduled query results. [#51781](https://github.com/elastic/beats/pull/51781)
* Rename log key `component` to `log.logger` to prevent mapping conflict on Elasticsearch. [#52912](https://github.com/elastic/beats/pull/52912) [#52888](https://github.com/elastic/beats/issues/52888)
* Stamp `space_id` on Osquerybeat live query result and profile documents so results are visible in non-default Kibana spaces. [#52915](https://github.com/elastic/beats/pull/52915) 

**Packetbeat**

* Censor HTTP parameters for mixed-case `hide_keywords` entries. [#52650](https://github.com/elastic/beats/pull/52650) 
* Fix bounds-check panics in Packetbeat TLS and Cassandra protocol parsers. [#52870](https://github.com/elastic/beats/pull/52870) 
* Fix network flow collection under the Elastic Agent OTel runtime. [#52932](https://github.com/elastic/beats/pull/52932) [#52931](https://github.com/elastic/beats/issues/52931)

