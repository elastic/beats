## 9.4.7 [beats-release-notes-9.4.7]



### Features and enhancements [beats-9.4.7-features-enhancements]


**All**

* Update Go to 1.26.7. [#52750](https://github.com/elastic/beats/pull/52750) 

**Filebeat**

* Add hints.input_allow_list to restrict Filebeat hints-generated input types. [#52869](https://github.com/elastic/beats/pull/52869) [#52624](https://github.com/elastic/beats/issues/52624)
* Report health status from the ETW input when running under Elastic Agent. [#53070](https://github.com/elastic/beats/pull/53070) [#44271](https://github.com/elastic/beats/issues/44271)

  The ETW input now reports Starting, Configuring, Running, Degraded and Failed
  states to Elastic Agent. Failure messages include the session name, the
  configured provider and the raw Windows error code, and access-denied errors
  spell out the privileges ETW sessions need. The input reports Degraded after
  `failure_threshold` consecutive unreadable events (default 3) and recovers
  after `recovery_threshold` consecutive good events (default 1). Set
  `failure_threshold` to 0 to disable Degraded reporting for unreadable events.
  

**Packetbeat**

* Log Npcap installer diagnostic files on installation failure. [#52667](https://github.com/elastic/beats/pull/52667) 


### Fixes [beats-9.4.7-fixes]


**Filebeat**

* Scope the aws-s3 polling state registry to the input&#39;s bucket. [#52728](https://github.com/elastic/beats/pull/52728) [#52721](https://github.com/elastic/beats/issues/52721)

  All aws-s3 inputs of a process share one persistent state registry. Each
  polling input loaded every input&#39;s states into memory, and its periodic
  registry cleanup then deleted from the shared store all entries missing
  from its own bucket listing, including the other inputs&#39; states. After a
  restart the affected inputs found no persisted state and re-ingested
  their buckets. Loading is now scoped to the input&#39;s bucket and key
  prefix. This also removes the duplicated memory usage of the registry
  when several polling inputs run in one process.
  
* Prevent httpjson chain pagination from advancing the cursor when a chained request fails. [#52996](https://github.com/elastic/beats/pull/52996) 

  When a paginated httpjson root request drives chain steps, a transient chain
  failure (timeout, 502, connection reset) was logged and ignored after the
  page cursor had already been committed. Later intervals then skipped the
  failed page&#39;s chain data. The interval now fails and the cursor stays on the
  last successfully chained page so the next run retries it.
  
* Follow every CrowdStrike discover resource instead of only the first feed. [#52997](https://github.com/elastic/beats/pull/52997) 

  FalconHose followSession iterated discovered resources but blocked in the
  first feed&#39;s decode loop and returned on EOF, so additional feeds in the same
  discover response were never read. Each resource is now followed
  concurrently with a merged per-feed cursor.
  

**Heartbeat**

* Match Kibana maintenance windows for nth-weekday, tzid, until, hourly, inclusive duration, and old dtstart. [#52572](https://github.com/elastic/beats/pull/52572) [#52571](https://github.com/elastic/beats/issues/52571)

**Metricbeat**

* Fix Kubernetes metadata watcher deadlock that blocked metricset shutdown. [#52409](https://github.com/elastic/beats/pull/52409) 

  The Kubernetes metadata enricher held the shared watcher registry lock while waiting for an informer cache to synchronize. When the Kubernetes API was unreachable or RBAC denied the initial LIST, that wait never completed and Stop() could not acquire the same lock to cancel it, hanging the metricset and blocking shutdown. Watcher startup now runs outside the registry lock as a shared attempt, and each metricset can cancel its own wait independently of other owners of the same watcher. Metricsets that successfully registered all their required metadata watchers now wait for those watchers to be ready before publishing. When add_metadata is disabled or client creation fails, the enricher is a no-op and publishing continues unenriched as before. A metricset whose extra metadata watcher (such as namespace enrichment) failed to register or synchronize does not collect.
  

**Osquerybeat**

* Propagate osquery live-query space IDs to action response events. [#50808](https://github.com/elastic/beats/pull/50808) 
* Stamp space_id on Osquerybeat live query result and profile documents so results are visible in non-default Kibana spaces. [#52915](https://github.com/elastic/beats/pull/52915) 

**Packetbeat**

* Censor HTTP parameters for mixed-case `hide_keywords` entries. [#52650](https://github.com/elastic/beats/pull/52650) 
* Fix bounds-check panics in Packetbeat TLS and Cassandra protocol parsers. [#52870](https://github.com/elastic/beats/pull/52870) 

