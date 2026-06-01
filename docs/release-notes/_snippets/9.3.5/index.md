## 9.3.5 [beats-release-notes-9.3.5]



### Features and enhancements [beats-9.3.5-features-enhancements]


**All**

* Update Go to v1.26.3. [#50644](https://github.com/elastic/beats/pull/50644) 

**Filebeat**

* Match `http.ServeMux` redirect status code for path cleaning in `http_endpoint` mux. [#50686](https://github.com/elastic/beats/pull/50686) 

**Libbeat**

* Cache `add_locale` processor and refresh only when `zone` or `offset` changes. [#50343](https://github.com/elastic/beats/pull/50343) 
* Update `ebpfevents` to v0.9.0. [#50609](https://github.com/elastic/beats/pull/50609) 

**Metricbeat**

* Add `elasticsearch/security_stats` metricset to the Elasticsearch module. [#50674](https://github.com/elastic/beats/pull/50674) 
* Migrate `azure/app_insights` metricset off the deprecated track-1 Azure SDK and `go-autorest`, and use `azcore` directly. [#50392](https://github.com/elastic/beats/pull/50392) 


### Fixes [beats-9.3.5-fixes]


**All**

* Update Go to v1.25.9. [#50049](https://github.com/elastic/beats/pull/50049) 
* Bump `aws-sdk-go-v2/service/cloudwatchlogs` to v1.65.0 to fix GHSA-xmrv-pmrh-hhx2. [#50215](https://github.com/elastic/beats/pull/50215) 
* Update `github.com/Azure/go-ntlmssp` to v0.1.1. [#50497](https://github.com/elastic/beats/pull/50497) 
* Initialize disk queue frame IDs from persisted state. [#50534](https://github.com/elastic/beats/pull/50534) 
* Fix OTel Beat processor to honor `when` conditions. [#50555](https://github.com/elastic/beats/pull/50555) 
* Fix race in pipeline client between `Publish` and `Close` that could skip waiting for events to be acknowledged. [#50625](https://github.com/elastic/beats/pull/50625) [#49390](https://github.com/elastic/beats/issues/49390)

**Auditbeat**

* Release the bolt file lock when the last datastore bucket is closed. [#50386](https://github.com/elastic/beats/pull/50386) [#50381](https://github.com/elastic/beats/issues/50381)

**Filebeat**

* Fix internal processing time metric for `azureeventhub` input. [#40547](https://github.com/elastic/beats/pull/40547) 
* Fix a race condition during multiline parser shutdown. [#49980](https://github.com/elastic/beats/pull/49980) 
* Re-evaluate `url_program` on each websocket reconnect using evolved cursor state. [#50383](https://github.com/elastic/beats/pull/50383) 
* Reduce allocation pressure in `httpjson` cursor update and split paths. [#50384](https://github.com/elastic/beats/pull/50384) 
* Fix Okta entity analytics OAuth2 config unpacking for `jwk_json` and `jwk_pem` fields. [#50406](https://github.com/elastic/beats/pull/50406) 
* Fix token refresh for `jwk_pem`/`jwk_file` in the `cel`, `httpjson`, and `okta` inputs. [#50433](https://github.com/elastic/beats/pull/50433) [#50426](https://github.com/elastic/beats/issues/50426)
* Fix Active Directory entity analytics to emit device attributes under `activedirectory.device`. [#50472](https://github.com/elastic/beats/pull/50472) [#50471](https://github.com/elastic/beats/issues/50471)
* Fix handling of OAuth2.0 timeouts in CrowdStrike streaming input. [#50492](https://github.com/elastic/beats/pull/50492) 
* Accept string values for `secret_state` to support Fleet secret resolution. [#50508](https://github.com/elastic/beats/pull/50508) 
* Respect `max_bytes`/`message_max_bytes` when reading the first chunk of CRI partial lines. [#50552](https://github.com/elastic/beats/pull/50552) 
* When Filestream is migrating the registry key from inputs that did not have an ID, match files by path and file identity instead of only path. [#50599](https://github.com/elastic/beats/pull/50599) 
* Fixes UDP input crashes on Windows when oversized datagrams are received. [#50770](https://github.com/elastic/beats/pull/50770) [#50718](https://github.com/elastic/beats/issues/50718)
* Make CrowdStrike streaming input cancel refresh goroutines on session reconnect. [#50803](https://github.com/elastic/beats/pull/50803) 

**Heartbeat**

* Upgrade npm to v11 in non-wolfi Heartbeat Docker images. [#50598](https://github.com/elastic/beats/pull/50598) 

**Libbeat**

* Fix bulk indexing failures that could occur when using the Elasticsearch output with some non-Elastic backends. [#49557](https://github.com/elastic/beats/pull/49557) [#49558](https://github.com/elastic/beats/issues/49558)
* Fix conversion of time duration fields such as `event.duration` when using Beat receivers. [#50302](https://github.com/elastic/beats/pull/50302) 
* Fix OTel map conversion for `[]time.Duration` fields to avoid dropping duration slices. [#50486](https://github.com/elastic/beats/pull/50486) 

**Metricbeat**

* Fix panic in Azure module when all configured resources match no Azure resources. [#50498](https://github.com/elastic/beats/pull/50498) 
* Elasticsearch module cluster state requests no longer append `local=true`. [#50723](https://github.com/elastic/beats/pull/50723) [#50722](https://github.com/elastic/beats/issues/50722)

**Winlogbeat**

* Fix `Long.decode` failures in the Painless script for the Windows security ingest pipeline. [#49869](https://github.com/elastic/beats/pull/49869) 
* Disable Winlogbeat record ID gap detection when using `xml_query` so filtered queries do not loop on non-contiguous record IDs. [#50443](https://github.com/elastic/beats/pull/50443) 

