## 9.5.3 [beats-release-notes-9.5.3]



### Features and enhancements [beats-9.5.3-features-enhancements]


**All**

* Update Go to 1.26.7. [#52750](https://github.com/elastic/beats/pull/52750) 

**Filebeat**

* Reduce Filestream memory use with multiple Filebeat receivers. [#52326](https://github.com/elastic/beats/pull/52326) 

  Filebeat now shares Filestream registry state between Filebeat receiver instances that use the same persistent registry. This avoids keeping a separate in-memory copy of the same state for each receiver, reducing memory use in workloads with a large number of inputs, like Kubernetes.
  

**Packetbeat**

* Log Npcap installer diagnostic files on installation failure. [#52667](https://github.com/elastic/beats/pull/52667) 


### Fixes [beats-9.5.3-fixes]


**Filebeat**

* Fix auditd filestream parser to emit &#34;unset&#34; for unset auid and ses fields. [#52614](https://github.com/elastic/beats/pull/52614) 
* Fix CEL input panic on nil rate-limit fields and 429 retry hot-loop. [#52684](https://github.com/elastic/beats/pull/52684) 
* Scope the aws-s3 polling state registry to the input&#39;s bucket. [#52728](https://github.com/elastic/beats/pull/52728) [#52721](https://github.com/elastic/beats/issues/52721)

  All aws-s3 inputs of a process share one persistent state registry. Each
  polling input loaded every input&#39;s states into memory, and its periodic
  registry cleanup then deleted from the shared store all entries missing
  from its own bucket listing, including the other inputs&#39; states. After a
  restart the affected inputs found no persisted state and re-ingested
  their buckets. Loading is now scoped to the input&#39;s bucket and key
  prefix. This also removes the duplicated memory usage of the registry
  when several polling inputs run in one process.
  
* Log entity analytics sync interruption by input shutdown or reconfiguration at info level instead of error, and no longer mark the input degraded for it. [#52950](https://github.com/elastic/beats/pull/52950) [#52945](https://github.com/elastic/beats/issues/52945)

  When an entity analytics sync was interrupted because the input&#39;s
  context was cancelled — agent shutdown, restart, or a policy change
  re-rendering the input mid-sync — the failure was logged as an error
  such as &#34;Error running full sync: ... context canceled&#34; and the
  input was marked Degraded. This routine lifecycle event is now
  logged at info level as a sync interruption and no longer degrades
  the input&#39;s status. All entity analytics providers
  (activedirectory, azuread, jamf, okta) and the minimal-state runner
  are covered. Other errors, including context deadline expiry, keep
  the previous error treatment.
  

**Osquerybeat**

* Emit pack_name and query_name in osquerybeat scheduled query results. [#51781](https://github.com/elastic/beats/pull/51781) 

  Scheduled pack queries (both osquery native interval schedules and osquerybeat
  RRULE schedules) now include two additional fields in their result and response
  documents: pack_name, taken from the new optional `pack_name` config field on a
  pack (alongside the existing pack_id), and query_name, taken from the pack&#39;s
  queries config map key. Both fields are only emitted when set. These fields are
  required by the Osquery Manager dashboards, which group and label scheduled
  query results by pack and query name; without them the dashboards cannot render
  those results correctly.
  
* Rename log key `component` to `log.logger` to prevent mapping conflict on Elasticsearch. [#52912](https://github.com/elastic/beats/pull/52912) [#52888](https://github.com/elastic/beats/issues/52888)
* Stamp space_id on Osquerybeat live query result and profile documents so results are visible in non-default Kibana spaces. [#52915](https://github.com/elastic/beats/pull/52915) 

**Packetbeat**

* Censor HTTP parameters for mixed-case `hide_keywords` entries. [#52650](https://github.com/elastic/beats/pull/52650) 
* Fix bounds-check panics in Packetbeat TLS and Cassandra protocol parsers. [#52870](https://github.com/elastic/beats/pull/52870) 
* Fix network flow collection under the Elastic Agent OTel runtime. [#52932](https://github.com/elastic/beats/pull/52932) [#52931](https://github.com/elastic/beats/issues/52931)

  Under the Elastic Agent OTel runtime, the flows stream is delivered in the
  protocols list, where Packetbeat dropped it as an unknown protocol plugin,
  collecting no flow events. Flow entries in the protocols list are now routed
  to the flows configuration, and unknown protocol entries mark Packetbeat as
  degraded instead of being silently ignored.
  

