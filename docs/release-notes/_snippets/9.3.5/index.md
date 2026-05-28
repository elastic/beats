## 9.3.5 [beats-release-notes-9.3.5]



### Features and enhancements [beats-9.3.5-features-enhancements]


**All**

* Update Go to 1.26.3. [#50644](https://github.com/elastic/beats/pull/50644) 

**Filebeat**

* Match http.ServeMux redirect status code for path cleaning in http_endpoint mux. [#50686](https://github.com/elastic/beats/pull/50686) 

**Libbeat**

* Cache add_locale processor and refresh only when zone or offset changes. [#50343](https://github.com/elastic/beats/pull/50343) 
* Update ebpfevents to 0.9.0. [#50609](https://github.com/elastic/beats/pull/50609) 

**Metricbeat**

* Add `elasticsearch/security_stats` metricset for the Elasticsearch module. [#50674](https://github.com/elastic/beats/pull/50674) 

  Adds a new `security_stats` metricset to the Elasticsearch module that
  scrapes the per-node `GET /_security/stats` endpoint (available since
  Elasticsearch 9.2). The first metric exposed is the Document Level Security
  cache (entries, memory, hits, misses, evictions, hit/miss latency), enabling
  fleet-wide observability of DLS cache health from Stack Monitoring. Each
  event is enriched with node name, roles, and version via a single
  filter-path-scoped /_nodes call per scrape so consumers can slice by node,
  role, or stack version without joining across data streams.
  
* Migrate azure/app_insights metricset off the deprecated track-1 Azure SDK and go-autorest, using azcore directly. [#50392](https://github.com/elastic/beats/pull/50392) 


### Fixes [beats-9.3.5-fixes]


**All**

* Update to Go 1.25.9. [#50049](https://github.com/elastic/beats/pull/50049) 
* Bump aws-sdk-go-v2/service/cloudwatchlogs to v1.65.0 to fix GHSA-xmrv-pmrh-hhx2. [#50215](https://github.com/elastic/beats/pull/50215) 
* Update github.com/Azure/go-ntlmssp to v0.1.1. [#50497](https://github.com/elastic/beats/pull/50497) 
* Initialize disk queue frame IDs from persisted state. [#50534](https://github.com/elastic/beats/pull/50534) 
* Fix OTel Beat processor to honor `when` conditions. [#50555](https://github.com/elastic/beats/pull/50555) 
* Fix race in pipeline client between Publish and Close that could skip waiting for event acks. [#50625](https://github.com/elastic/beats/pull/50625) [#49390](https://github.com/elastic/beats/issues/49390)

**Auditbeat**

* Release the bolt file lock when the last datastore bucket is closed. [#50386](https://github.com/elastic/beats/pull/50386) [#50381](https://github.com/elastic/beats/issues/50381)

**Filebeat**

* Filestream only reports degraded status for permanent harvester errors. [#49481](https://github.com/elastic/beats/pull/49481) [#49451](https://github.com/elastic/beats/issues/49451)
* Fix internal processing time metric for azureeventhub input. [#40547](https://github.com/elastic/beats/pull/40547) 
* Fix race condition during multiline parser shutdown. [#49980](https://github.com/elastic/beats/pull/49980) 
* Re-evaluate url_program on each websocket reconnect using evolved cursor state. [#50383](https://github.com/elastic/beats/pull/50383) 

  The streaming input now re-evaluates url_program before each websocket
  reconnection (both error recovery and OAuth2 token refresh), allowing
  cursor state accumulated during the session to influence the reconnect URL.
  Previously url_program was evaluated once at startup and the result was
  reused for all subsequent connections. The process function also now returns
  the evolved cursor so that callers can propagate it into the shared state.
  
* Reduce allocation pressure in httpjson cursor update and split paths. [#50384](https://github.com/elastic/beats/pull/50384) 
* Fix Okta entity analytics OAuth2 config unpacking for jwk_json and jwk_pem fields. [#50406](https://github.com/elastic/beats/pull/50406) 
* Fix token refresh for jwk_pem/jwk_file in cel, httpjson, okta inputs. [#50433](https://github.com/elastic/beats/pull/50433) [#50426](https://github.com/elastic/beats/issues/50426)
* Fix Active Directory entity analytics to emit device attributes under activedirectory.device. [#50472](https://github.com/elastic/beats/pull/50472) [#50471](https://github.com/elastic/beats/issues/50471)
* Fix handling of OAuth2.0 timeouts in CrowdStrike streaming input. [#50492](https://github.com/elastic/beats/pull/50492) 
* Accept string values for secret_state to support Fleet secret resolution. [#50508](https://github.com/elastic/beats/pull/50508) 
* Respect max_bytes/message_max_bytes when reading the first chunk of CRI partial lines. [#50552](https://github.com/elastic/beats/pull/50552) 
* When Filestream is migrating the registry key from inputs that did
not have an ID, match files by path and file identity instead of
only path. Matching only by path could lead to rotated/moved files
having the wrong offset.
. [#50599](https://github.com/elastic/beats/pull/50599) 
* Fixes UDP input crashes on Windows when oversized datagrams are received. [#50770](https://github.com/elastic/beats/pull/50770) [#50718](https://github.com/elastic/beats/issues/50718)
* Make CrowdStrike streaming input cancel refresh goroutines on session reconnect. [#50803](https://github.com/elastic/beats/pull/50803) 

**Heartbeat**

* Upgrade npm to v11 in non-wolfi heartbeat Docker images. [#50598](https://github.com/elastic/beats/pull/50598) 

**Libbeat**

* Fix bulk indexing failures that could occur when using the Elasticsearch output with some non-Elastic backends. [#49557](https://github.com/elastic/beats/pull/49557) [#49558](https://github.com/elastic/beats/issues/49558)
* Fix conversion of time duration fields such as event.duration when using Beats receivers. [#50302](https://github.com/elastic/beats/pull/50302) 
* Fix OTel map conversion for []time.Duration fields to avoid dropping duration slices. [#50486](https://github.com/elastic/beats/pull/50486) 

**Metricbeat**

* Fix panic in azure module when all configured resources match no Azure resources. [#50498](https://github.com/elastic/beats/pull/50498) 
* Elasticsearch module cluster state requests no longer append local=true. [#50723](https://github.com/elastic/beats/pull/50723) [#50722](https://github.com/elastic/beats/issues/50722)

**Winlogbeat**

* Fix Long decoding error in painless script for windows ingest pipeline. [#49869](https://github.com/elastic/beats/pull/49869) 
* Disable Winlogbeat record ID gap detection when using xml_query so filtered queries do not loop on non-contiguous record IDs. [#50443](https://github.com/elastic/beats/pull/50443) 

