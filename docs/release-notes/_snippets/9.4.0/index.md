## 9.4.0 [beats-release-notes-9.4.0]



### Features and enhancements [beats-9.4.0-features-enhancements]


**All**

* Export all Beat receiver metrics to OTel telemetry. [#49300](https://github.com/elastic/beats/pull/49300) 
* Add `add_agent_metadata` processor to inject agent metadata efficiently. [#49667](https://github.com/elastic/beats/pull/49667) 
* Update OTel Collector components to v0.149.0/v1.55.0. [#50057](https://github.com/elastic/beats/pull/50057) 

**Elastic agent**

* Logstash exporter now reports accurate error status to EDOT. [#49169](https://github.com/elastic/beats/pull/49169) 

**Filebeat**

* Add a lexicographical polling mode to the AWS-S3 input. [#48310](https://github.com/elastic/beats/pull/48310) [#47926](https://github.com/elastic/beats/issues/47926)
* Optimize opening files in `filestream` for better performance. [#48506](https://github.com/elastic/beats/pull/48506) 
* Instruments the CEL input with OpenTelemetry tracing. [#48440](https://github.com/elastic/beats/pull/48440) 
* Add ES state store routing for lexicographical mode. [#48944](https://github.com/elastic/beats/pull/48944) 
* Add an experimental bbolt-based registry backend for Filebeat. [#48879](https://github.com/elastic/beats/pull/48879) 
* Add MFA enrichment support to the `azure-ad` entity analytics provider. [#49843](https://github.com/elastic/beats/pull/49843) 
* Add a `querylog` fileset to the Filebeat `elasticsearch` module for NDJSON query logs. [#49914](https://github.com/elastic/beats/pull/49914) [#43622](https://github.com/elastic/beats/issues/43622)
* Optimize `filestream` to allocate less memory when applying include/exclude line filters. [#49013](https://github.com/elastic/beats/pull/49013) 
* Add capacity to collect empty Active Directory groups to entity analytics input. [#49093](https://github.com/elastic/beats/pull/49093) 
* Add input redirection support through a new Redirector mechanism. [#49613](https://github.com/elastic/beats/pull/49613) 
* Allow HTTP JSON input redirection to the CEL input. [#49614](https://github.com/elastic/beats/pull/49614) 
* Update mito CEL library to v1.25.1 and cel-go runtime to v0.27.0. [#49683](https://github.com/elastic/beats/pull/49683) 
* Add `perms` enrichment option to the Okta entity analytics provider to collect permissions for custom roles. [#49805](https://github.com/elastic/beats/pull/49805) [#49779](https://github.com/elastic/beats/issues/49779)
* Add a `devices` enrichment option to the Okta entity analytics provider. [#49813](https://github.com/elastic/beats/pull/49813) [#49780](https://github.com/elastic/beats/issues/49780)
* Add a `supervises` enrichment option to the Okta entity analytics provider. [#49825](https://github.com/elastic/beats/pull/49825) [#49781](https://github.com/elastic/beats/issues/49781)

**Filebeat, metricbeat**

* Add `NewFactoryWithSettings` for Beat receivers to provide default home and path directories. [#49327](https://github.com/elastic/beats/pull/49327) [#11734](https://github.com/elastic/elastic-agent/issues/11734)

**Heartbeat**

* Add custom policy hashing and live-update functionality to integrations, allowing parameters to be updated without restarting the monitor. [#49326](https://github.com/elastic/beats/pull/49326) [#47511](https://github.com/elastic/beats/issues/47511)

**Metricbeat**

* Add cursor-based incremental data fetching to the SQL module query metricset. [#48722](https://github.com/elastic/beats/pull/48722) 
* Add `switchport_statuses` config option to filter Meraki switchports by status. [#47993](https://github.com/elastic/beats/pull/47993) 
* Add the `subexpiry` field to the Redis INFO Keyspace (Redis ≥ 7.4). [#47971](https://github.com/elastic/beats/pull/47971) [#26555](https://github.com/elastic/enhancements/issues/26555)
* Add Redis 6.0/7.0 info fields and deprecate `used_memory_lua`. [#48246](https://github.com/elastic/beats/pull/48246) 
* Add `state` field to IPSec tunnel metrics in `panw` module. [#48403](https://github.com/elastic/beats/pull/48403) 
* Map Docker network metrics to different types for better usability. [#47792](https://github.com/elastic/beats/pull/47792) 
* Add `observer.hostname` field to `panw` module. [#48825](https://github.com/elastic/beats/pull/48825) 
* Bump azure-sdk-for-go `armmonitor` from v0.8.0 to v0.11.0. [#49866](https://github.com/elastic/beats/pull/49866) 

**Osquerybeat**

* Add an `elastic_jumplists` table to the Osquery extension. [#47759](https://github.com/elastic/beats/pull/47759) 
* Add a code generator for creating typed Go packages from YAML table/view specifications. [#48533](https://github.com/elastic/beats/pull/48533)
* Add automatic jumplists to the `elastic_jumplists` table. [#48032](https://github.com/elastic/beats/pull/48032) 
* Add `elastic_host_processes` and `elastic_host_users` tables and `host_processes` and `host_users` views. [#48794](https://github.com/elastic/beats/pull/48794) 
* Add optional query profiling for scheduled and live Osquery runs. [#49514](https://github.com/elastic/beats/pull/49514) 
* Updates Osquerybeat filters to avoid reliance on type assertion. [#48540](https://github.com/elastic/beats/pull/48540) 
* Allow for passing of osqueryd client to extension tables. [#48544](https://github.com/elastic/beats/pull/48544) 
* Improve gentables and add `elastic_browser_history` specification with registry integration. [#48733](https://github.com/elastic/beats/pull/48733) 
* Migrate `elastic_file_analysis` table to Osquery table specification and a dedicated package. [#48774](https://github.com/elastic/beats/pull/48774) 
* Add `elastic_host_groups` table, `host_groups` view, and default view hooks in generator. [#48775](https://github.com/elastic/beats/pull/48775) 
* Add `amcache` table, view specifications and generated tables/view for Osquery extension. [#48802](https://github.com/elastic/beats/pull/48802) 
* Add native scheduled query metadata and `schedule_id` correlation fields. [#49040](https://github.com/elastic/beats/pull/49040) 
* Add `elastic_jumplists` specification integration and harden Osquerybeat generation workflow. [#49058](https://github.com/elastic/beats/pull/49058) 
* Add support for per-platform custom osqueryd artifact install in Osquerybeat. [#49306](https://github.com/elastic/beats/pull/49306) [#48955](https://github.com/elastic/beats/issues/48955)


### Fixes [beats-9.4.0-fixes]


**Agentbeat**

* Update transient dependency `github.com/go-jose/go-jose/v4` to v4.1.4. [#49975](https://github.com/elastic/beats/pull/49975) 

**All**

* Update Go to v1.25.9. [#50049](https://github.com/elastic/beats/pull/50049) 
* Bump `aws-sdk-go-v2/service/cloudwatchlogs` to v1.65.0 to fix GHSA-xmrv-pmrh-hhx2. [#50215](https://github.com/elastic/beats/pull/50215) 

**Filebeat**

* Support `abuse.ch` auth key usage in the Threat Intel module. [#45212](https://github.com/elastic/beats/pull/45212) [#45206](https://github.com/elastic/beats/issues/45206)
* Fix `max_body_bytes` setting not working without HMAC and add missing documentation configuration options in HTTP Endpoint input. [#48550](https://github.com/elastic/beats/pull/48550) [#48512](https://github.com/elastic/beats/issues/48512)

  Previously, the max_body_bytes setting was only applied during HMAC validation, meaning it had no effect on requests that didn&#39;t use HMAC authentication.
  This fix ensures that body size limiting is applied to all incoming requests regardless of authentication method.
  Additionally, restored missing documentation for the max_body_bytes setting in the HTTP Endpoint input.
  
* Fix `http_endpoint` input shared server lifecycle causing joiner deadlock and creator killing unrelated inputs. [#49415](https://github.com/elastic/beats/pull/49415) 
* Fix a typo in CEL input OTel tracing logging. [#49692](https://github.com/elastic/beats/pull/49692) [#49625](https://github.com/elastic/beats/issues/49625)
* Fix the `container` input not respecting `max_bytes` when parsing CRI partial lines. [#49743](https://github.com/elastic/beats/pull/49743) [#49259](https://github.com/elastic/beats/issues/49259)
* Fix internal processing time metric for `azureeventhub` input. [#40547](https://github.com/elastic/beats/pull/40547) 
* Fix CSV decoder producing malformed JSON when field values contain double quotes in `azure-blob-storage` input. [#50097](https://github.com/elastic/beats/pull/50097) 
* Update `cel-go` to v0.28.0, fixing runtime error location reporting. [#50176](https://github.com/elastic/beats/pull/50176) 
* Re-evaluate `url_program` on each websocket reconnect using evolved cursor state. [#50383](https://github.com/elastic/beats/pull/50383) 
* Reduce allocation pressure in `httpjson` cursor update and split paths. [#50384](https://github.com/elastic/beats/pull/50384) 

**Libbeat**

* Fix conversion of time duration fields such as `event.duration` when using Beat receivers. [#50302](https://github.com/elastic/beats/pull/50302) 

**Metricbeat**

* AutoOps ES module update to use UUID v7 without dashes to reduce payloads. [#50078](https://github.com/elastic/beats/pull/50078) 

**Osquerybeat**

* Fix jumplist table to ensure embedded fields are exported. [#49649](https://github.com/elastic/beats/pull/49649) 
* Avoid mutating osquery install config during validation to prevent races. [#49769](https://github.com/elastic/beats/pull/49769) 

**Packetbeat**

* Fix janitor goroutine leaks and decoder cleanup lifecycle on route changes. [#48836](https://github.com/elastic/beats/pull/48836) 

**Winlogbeat**

* Fix `no_more_events` stop losing final batch of events when `io.EOF` is returned alongside records. [#49012](https://github.com/elastic/beats/pull/49012) [#47388](https://github.com/elastic/beats/issues/47388)

