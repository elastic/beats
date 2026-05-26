## 9.4.2 [beats-release-notes-9.4.2]



### Features and enhancements [beats-9.4.2-features-enhancements]


**Filebeat**

* Match http.ServeMux redirect status code for path cleaning in http_endpoint mux. [#50686](https://github.com/elastic/beats/pull/50686) 

**Libbeat**

* Update ebpfevents to 0.9.0. [#50609](https://github.com/elastic/beats/pull/50609) 

**Metricbeat**

* Add failure_store metric to the stats metricset in the beat module. [#49452](https://github.com/elastic/beats/pull/49452) 
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


### Fixes [beats-9.4.2-fixes]


**All**

* Initialize disk queue frame IDs from persisted state. [#50534](https://github.com/elastic/beats/pull/50534) 
* Fix race in pipeline client between Publish and Close that could skip waiting for event acks. [#50625](https://github.com/elastic/beats/pull/50625) [#49390](https://github.com/elastic/beats/issues/49390)

**Auditbeat**

* Release the bolt file lock when the last datastore bucket is closed. [#50386](https://github.com/elastic/beats/pull/50386) [#50381](https://github.com/elastic/beats/issues/50381)

**Filebeat**

* Fix token refresh for jwk_pem/jwk_file in cel, httpjson, okta inputs. [#50433](https://github.com/elastic/beats/pull/50433) [#50426](https://github.com/elastic/beats/issues/50426)
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

**Metricbeat**

* Fix panic in azure module when all configured resources match no Azure resources. [#50498](https://github.com/elastic/beats/pull/50498) 
* Elasticsearch module cluster state requests no longer append local=true. [#50723](https://github.com/elastic/beats/pull/50723) [#50722](https://github.com/elastic/beats/issues/50722)

