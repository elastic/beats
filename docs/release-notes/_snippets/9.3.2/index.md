## 9.3.2 [beats-release-notes-9.3.2]



### Features and enhancements [beats-9.3.2-features-enhancements]


**Elastic agent**

* Fix a bug that could report stopped inputs as still running. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)

**Filebeat**

* Add optional token_url support for JWT Bearer Flow in Salesforce input. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#43963](https://github.com/elastic/beats/issues/43963)

  The Salesforce input now supports a separate `token_url` configuration for JWT Bearer Flow
  authentication. This allows users with custom Salesforce domains or restrictions on default
  endpoints (login.salesforce.com/test.salesforce.com) to specify a different token endpoint
  URL while keeping the audience URL separate. If token_url is not provided, the existing
  behavior of using the audience URL as the token endpoint is maintained.
  
* Empty files are excluded from processing in filestream as early as possible. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)

**Metricbeat**

* Add zswap compressed swap cache metrics to system memory metricset. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)
* Addition of Elasticsearch index mode and codec settings in Metricbeat index stats module. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)
* Add cgroupv2 CPU metrics to system.process dataset. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)
* Add swap field to system.process.memory metric set in metricbeat. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)
* Tbs-monitoring. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)

  Add new TBS metrics to monitor mappings
* Add a config to improve wildcard handling to report actual object names. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)
* Read Kibana status response body on 503 so monitoring captures the reason for outage. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)

**Packetbeat**

* Improves resiliency of the AMQP parser against invalid or corrupt data frames. [#48033](https://github.com/elastic/beats/pull/48033) 
* Bump bundled Windows Npcap OEM installer to v1.87. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)

**Winlogbeat**

* Move winlog filtering to Go-side evaluation and harden recovery paths. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)

  Winlogbeat and Filebeat winlog input now subscribe with unfiltered queries for non-custom configurations
  and apply ignore_older, provider, event_id, and level filtering in code. This avoids unreliable Windows
  query-filter behavior in affected environments while preserving custom xml_query passthrough. The change
  also improves read/iterator recovery behavior, keeps final-batch publish semantics on EOF, and adds a
  retry circuit-breaker for persistent render failures without partial events.
  


### Fixes [beats-9.3.2-fixes]


**All**

* Update elastic-agent-system-metrics to v0.14.0. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)
* Update elastic-agent-autodiscover to v0.10.2. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)
* Update elastic-agent-libs to v0.32.2. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)
* Update OpenTelemetry SDK to version v1.40.0. [#49126](https://github.com/elastic/beats/pull/49126) 
* Improve append processor behavior when merging values and removing duplicates. [#49021](https://github.com/elastic/beats/pull/49021) [#49020](https://github.com/elastic/beats/issues/49020)

  The append processor now appends values more consistently, avoiding nested
  entries in the target field. Duplicate removal is also more reliable, reducing
  processing errors and keeping output stable.
  
* Kafka client will avoid having more than a single metadata request to each broker in-flight at any given time. [#49307](https://github.com/elastic/beats/pull/49307) [#49210](https://github.com/elastic/beats/issues/49210)

**Beatreceiver**

* Fix reporting of beatreceiver 30s metrics. [#49236](https://github.com/elastic/beats/pull/49236) 

**Filebeat**

* Improve in-flight byte accounting in the HTTP Endpoint input. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#48456](https://github.com/elastic/beats/issues/48456)
* Honor non-fingerprint file_identity defaults in filestream. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)
* Fix handling of Crowdstrike streaming input state in retryable errors. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)
* Fix incremental group updates in Active Directory entity analytics provider. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)
* Demote missing user/device state lookup to debug log in Azure entity analytics provider. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)
* Fix CrowdStrike streaming session refresh scheduling to avoid tight refresh loops. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#49158](https://github.com/elastic/beats/issues/49158)

**Metricbeat**

* Update transient dependency filippo.io/edwards25519 to v1.1.1. [#49070](https://github.com/elastic/beats/pull/49070) 

**Osquerybeat**

* Update osquery-go dependency to v0.0.0-20260226222546-0cc22f415e57. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#2418](https://github.com/elastic/beats/issues/2418) [#47511](https://github.com/elastic/beats/issues/47511) [#48152](https://github.com/elastic/beats/issues/48152)

**Winlogbeat**

* Restore suppression of repeated channel-not-found open errors in winlogbeat eventlog runner. [#49476](https://github.com/elastic/beats/pull/49476) [#48880](https://github.com/elastic/beats/pull/48880) [#49521](https://github.com/elastic/beats/pull/49521) [#49326](https://github.com/elastic/beats/pull/49326) [#49445](https://github.com/elastic/beats/pull/49445) [#48979](https://github.com/elastic/beats/issues/48979)

  Reintroduces channel-not-found retry log suppression that was lost during the eventlog runner refactor.
  The first channel-not-found open error is logged at WARN, subsequent retries are logged at DEBUG, and
  the suppression state is reset after a successful open. This prevents repeated WARN/ERROR log noise
  when a configured channel is missing.
  

