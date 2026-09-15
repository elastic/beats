## 9.4.7 [beats-release-notes-9.4.7]



### Features and enhancements [beats-9.4.7-features-enhancements]


**All**

* Update Go to 1.26.7. [#52750](https://github.com/elastic/beats/pull/52750) 

**Filebeat**

* Add `hints.input_allow_list` to restrict Filebeat hints-generated input types. [#52869](https://github.com/elastic/beats/pull/52869) [#52624](https://github.com/elastic/beats/issues/52624)
* Report health status from the ETW input when running under Elastic Agent. [#53070](https://github.com/elastic/beats/pull/53070) [#44271](https://github.com/elastic/beats/issues/44271)
  

**Packetbeat**

* Log Npcap installer diagnostic files on installation failure. [#52667](https://github.com/elastic/beats/pull/52667) 


### Fixes [beats-9.4.7-fixes]


**Filebeat**

* Scope the `aws-s3` polling state registry to the input's bucket. [#52728](https://github.com/elastic/beats/pull/52728) [#52721](https://github.com/elastic/beats/issues/52721)
* Prevent `httpjson` chain pagination from advancing the cursor when a chained request fails. [#52996](https://github.com/elastic/beats/pull/52996)  
* Follow every CrowdStrike discover resource instead of only the first feed. [#52997](https://github.com/elastic/beats/pull/52997)
  

**Heartbeat**

* Align maintenance window recurrence handling with Kibana, including nth-weekday rules, `tzid` timezone evaluation, `until` end bounds, hourly recurrences, inclusive duration behavior, and legacy `dtstart` formats. [#52572](https://github.com/elastic/beats/pull/52572) [#52571](https://github.com/elastic/beats/issues/52571)

**Metricbeat**

* Fix Kubernetes metadata watcher deadlock that blocked metricset shutdown. [#52409](https://github.com/elastic/beats/pull/52409)
  

**Osquerybeat**

* Propagate Osquery live-query space IDs to action response events. [#50808](https://github.com/elastic/beats/pull/50808) 
* Stamp `space_id` on Osquerybeat live query result and profile documents so results are visible in non-default Kibana spaces. [#52915](https://github.com/elastic/beats/pull/52915) 

**Packetbeat**

* Censor HTTP parameters for mixed-case `hide_keywords` entries. [#52650](https://github.com/elastic/beats/pull/52650) 
* Fix bounds-check panics in Packetbeat TLS and Cassandra protocol parsers. [#52870](https://github.com/elastic/beats/pull/52870) 

