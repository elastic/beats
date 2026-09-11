## 9.5.4 [beats-release-notes-9.5.4]



### Features and enhancements [beats-9.5.4-features-enhancements]


**All**

* Add hostname overrides for Beat events. [#53029](https://github.com/elastic/beats/pull/53029) [#48923](https://github.com/elastic/beats/issues/48923)

  The hostname can be overridden with the --hostname flag or the top-level hostname config field. The flag wins when both are set. The config field also works for beats running as OTel receivers, which never see CLI flags. The add_host_metadata and add_observer_metadata processors honor the override: host.name reflects the override value, and observer.hostname does too when add_observer_metadata is in the pipeline. The override value is trimmed of whitespace; casing is preserved as supplied.

**Filebeat**

* Add hints.input_allow_list to restrict Filebeat hints-generated input types. [#53006](https://github.com/elastic/beats/pull/53006) [#52624](https://github.com/elastic/beats/issues/52624)
* Report health status from the ETW input when running under Elastic Agent. [#53089](https://github.com/elastic/beats/pull/53089) [#44271](https://github.com/elastic/beats/issues/44271)

  The ETW input now reports Starting, Configuring, Running, Degraded and Failed
  states to Elastic Agent. Failure messages include the session name, the
  configured provider and the raw Windows error code, and access-denied errors
  spell out the privileges ETW sessions need. The input reports Degraded after
  `failure_threshold` consecutive unreadable events (default 3) and recovers
  after `recovery_threshold` consecutive good events (default 1). Set
  `failure_threshold` to 0 to disable Degraded reporting for unreadable events.
  

**Metricbeat**

* Add ingest volume enrichment fields to autoops_es module. [#52914](https://github.com/elastic/beats/pull/52914) [#48923](https://github.com/elastic/beats/issues/48923)
* Collect logical_availability_zone and instance_configuration node attributes in the autoops_es node_stats metricset. [#53119](https://github.com/elastic/beats/pull/53119) [#48923](https://github.com/elastic/beats/issues/48923)
* Collect the frozen flood stage disk watermark in AutoOps cluster settings. [#53114](https://github.com/elastic/beats/pull/53114) [#52341](https://github.com/elastic/beats/issues/52341)


### Fixes [beats-9.5.4-fixes]


**Filebeat**

* Wire Sarama client logs in the Filebeat Kafka input so consumer-group lifecycle is visible. [#52980](https://github.com/elastic/beats/pull/52980) [#51305](https://github.com/elastic/beats/issues/51305)
* Prevent httpjson chain pagination from advancing the cursor when a chained request fails. [#53063](https://github.com/elastic/beats/pull/53063) [#48923](https://github.com/elastic/beats/issues/48923)

  When a paginated httpjson root request drives chain steps, a transient chain
  failure (timeout, 502, connection reset) was logged and ignored after the
  page cursor had already been committed. Later intervals then skipped the
  failed page&#39;s chain data. The interval now fails and the cursor stays on the
  last successfully chained page so the next run retries it.
  
* Follow every CrowdStrike discover resource instead of only the first feed. [#53060](https://github.com/elastic/beats/pull/53060) [#48923](https://github.com/elastic/beats/issues/48923)

  FalconHose followSession iterated discovered resources but blocked in the
  first feed&#39;s decode loop and returned on EOF, so additional feeds in the same
  discover response were never read. Each resource is now followed
  concurrently with a merged per-feed cursor.
  
* Update github.com/elastic/entcollect to fix bugs in the EntraID and Okta providers. [#52971](https://github.com/elastic/beats/pull/52971) [#48923](https://github.com/elastic/beats/issues/48923)

**Heartbeat**

* Match Kibana maintenance windows for nth-weekday, tzid, until, hourly, inclusive duration, and old dtstart. [#52957](https://github.com/elastic/beats/pull/52957) [#52571](https://github.com/elastic/beats/issues/52571)

**Metricbeat**

* Fix Kubernetes metadata watcher deadlock that blocked metricset shutdown. [#53040](https://github.com/elastic/beats/pull/53040) [#48923](https://github.com/elastic/beats/issues/48923)

  The Kubernetes metadata enricher held the shared watcher registry lock while waiting for an informer cache to synchronize. When the Kubernetes API was unreachable or RBAC denied the initial LIST, that wait never completed and Stop() could not acquire the same lock to cancel it, hanging the metricset and blocking shutdown. Watcher startup now runs outside the registry lock as a shared attempt, and each metricset can cancel its own wait independently of other owners of the same watcher. Metricsets that successfully registered all their required metadata watchers now wait for those watchers to be ready before publishing. When add_metadata is disabled or client creation fails, the enricher is a no-op and publishing continues unenriched as before. A metricset whose extra metadata watcher (such as namespace enrichment) failed to register or synchronize does not collect.
  

**Osquerybeat**

* Restart osqueryd instead of terminating osquerybeat when the extension manager ping times out. [#52993](https://github.com/elastic/beats/pull/52993) [#48923](https://github.com/elastic/beats/issues/48923)

  The osquery-go extension manager watchdog shares a single thrift socket, and a
  single 200ms lock, with extension registration and deregistration. Losing that
  lock race surfaces as &#34;extension ping failed: timeout after 200ms&#34;, which is
  indistinguishable from osqueryd actually being gone.
  
  That error was not in osquerybeat&#39;s recoverable set, so it propagated out of
  Run. As a supervised process the agent restarted the component and suppressed
  the failed state, making this self-healing and effectively invisible. Running
  as a beat receiver there is no such restart: the error is reported as
  status.Failed, maps to an OTel PermanentError, and the osquery_manager unit
  stays FAILED until the whole agent is restarted.
  
  Extension ping failures and network timeouts are now treated as recoverable,
  so the runner replays the last known inputs and restarts osqueryd.
  

