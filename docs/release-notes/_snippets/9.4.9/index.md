## 9.4.9 [beats-release-notes-9.4.9]



### Features and enhancements [beats-9.4.9-features-enhancements]


**All**

* Update Go to 1.26.10. [#53706](https://github.com/elastic/beats/pull/53706) 

**Filebeat**

* Ensure that the CEL input&#39;s events from previous evaluations are removed from the CEL state before starting a new evaluation. [#53817](https://github.com/elastic/beats/pull/53817) 
* Add proxy_url and proxy_disable support to the azure-eventhub input, reject websocket transport with processor v1, and propagate checkpoint migration errors. [#53534](https://github.com/elastic/beats/pull/53534) 

**Metricbeat**

* Collect the serverless.search.min_search_vcu cluster setting as a number in the AutoOps ES cluster_settings metricset. [#53943](https://github.com/elastic/beats/pull/53943) 
* Add os.cgroup.cpu.usage_percent to autoops_es node_stats metricset. [#53140](https://github.com/elastic/beats/pull/53140) [#52616](https://github.com/elastic/beats/issues/52616)

**Packetbeat**

* Bump npcap OEM installer to 1.88. [#53773](https://github.com/elastic/beats/pull/53773) 

  Bump the npcap OEM installer version used by Packetbeat to 1.88.
* Bump npcap OEM installer to 1.89. [#53706](https://github.com/elastic/beats/pull/53706) 

  Bump the npcap OEM installer version used by Packetbeat to 1.89.


### Fixes [beats-9.4.9-fixes]


**All**

* Fix data race in translate_ldap_attribute. [#51690](https://github.com/elastic/beats/pull/51690) 
* Only log metrics namespaces listed in logging.metrics.namespaces (default [stats]), stops logging per-input metrics (monitoring.dataset.*) by default. [#53919](https://github.com/elastic/beats/pull/53919) 

**Auditbeat**

* Fix the file_integrity eBPF backend failing to start on Linux 6.15 and later. [#53511](https://github.com/elastic/beats/pull/53511) [#46719](https://github.com/elastic/beats/issues/46719)

  Update github.com/elastic/ebpfevents to v0.11.0. On Linux 6.15 and later the
  file_integrity module&#39;s eBPF backend failed to load its probes with
  &#34;bad CO-RE relocation&#34; (tty_driver and kernfs_node changed in 6.15), so the
  module never started. File access, modification and change timestamps are
  also reported correctly again on Linux 6.11 and later. Programs that are not
  used on the running kernel are no longer loaded, which shortens startup. When
  the eBPF loader has to fall back because some probes fail to load on the
  running kernel, the module logs a warning, and it fails to start if any file
  event type it needs is disabled.
  
* Fix a panic in the file_integrity eBPF backend when its eBPF loader fails to start. [#53785](https://github.com/elastic/beats/pull/53785) 

  The eBPF backend loaded its probes once to check that they work, and again when
  the module started. If the first load succeeded but the second failed, Auditbeat
  panicked with a nil pointer dereference. The probes are now loaded only when the
  module starts; if that fails, the module fails to start with the loader&#39;s error
  and a later start tries again.
  
* Fix file_integrity kprobes backend hanging at startup when probe verification keeps failing. [#53782](https://github.com/elastic/beats/pull/53782) 

**Elastic agent**

* Fix unbounded memory growth when the Logstash exporter admits batches out of order under the OTel runtime. [#54036](https://github.com/elastic/beats/pull/54036) 

**Filebeat**

* Update github.com/apache/thrift to v0.25.0 to fix CVE-2026-82458 in Parquet decoding. [#53636](https://github.com/elastic/beats/pull/53636) 
* Fix cursor inputs losing their saved position when two inputs start at the same time. [#53204](https://github.com/elastic/beats/pull/53204) 
* Resume CrowdStrike Event Streams after dropped firehose connections without exhausting retries. [#53416](https://github.com/elastic/beats/pull/53416) 

  The CrowdStrike streaming input counted a firehose connection dropped mid-stream
  (reported as &#34;error decoding event: unexpected EOF&#34;) as a failed attempt, and only
  reset the count when a stream ended cleanly. Drops spread over hours of healthy
  streaming accumulated until max_attempts was exceeded and the input stopped until
  it was restarted. A drop after events were delivered now reconnects from the cursor
  and resets the attempt count, the unit reports healthy again once events flow, and
  feed connects, ends, interruptions and non-2xx session refresh responses are logged.
  Each session error is now counted once in errors_total, and a clean end of stream is
  not counted as an error.
  
* HTTP Endpoint options_headers must be non-empty and options_response_code must be valid when options_headers is configured. A previously accepted empty options_headers: {} now fails at startup.
. [#53417](https://github.com/elastic/beats/pull/53417) 
* Fix azure-eventhub v1 input crashing the otel collector with concurrent map iteration and map write. [#53822](https://github.com/elastic/beats/pull/53822) [#53816](https://github.com/elastic/beats/issues/53816)
* Fix TCP and UDP input receive queue and packet drop metrics always reporting 0 when listening on 0.0.0.0 on hosts with IPv6 enabled. [#53972](https://github.com/elastic/beats/pull/53972) 
* Fix system/auth module journald ingest pipeline failing on events without journald.process.name. [#54033](https://github.com/elastic/beats/pull/54033) [#54013](https://github.com/elastic/beats/issues/54013)
* Reduce Azure Event Hub v2 memory retention while waiting for acknowledgments. [#54014](https://github.com/elastic/beats/pull/54014) 

**Heartbeat**

* Bump npm to 12.2.1 and Node.js to 22.23.4 (24.22.1 on Wolfi) in the Heartbeat Docker images to remove vulnerable undici copies. [#53947](https://github.com/elastic/beats/pull/53947) 
* Fix malformed Synthetics runner log messages. [#53593](https://github.com/elastic/beats/pull/53593) 
* Fix Heartbeat memory growth by releasing popped tasks from the scheduler timer queue. [#53664](https://github.com/elastic/beats/pull/53664) [#53662](https://github.com/elastic/beats/issues/53662)

**Metricbeat**

* Skip unsupported Azure Monitor metric namespaces when ignore_unsupported is true instead of failing collection on HTTP 400. [#53643](https://github.com/elastic/beats/pull/53643) [#52999](https://github.com/elastic/beats/issues/52999)
* Collect running snapshot tasks in the autoops_es tasks_management metricset by filtering on cluster:admin/snapshot instead of the nonexistent indices:admin/snapshot actions. [#54100](https://github.com/elastic/beats/pull/54100) 

**Osquerybeat**

* Restore osquery-extension compatibility with Linux systems on glibc before 2.28. [#52651](https://github.com/elastic/beats/pull/52651) 

  Building SQLite with SQLITE_DISABLE_LFS on 64-bit Linux avoids a dependency
  on glibc 2.28. Functionality is unchanged but compatibility is restored for
  glibc 2.27 systems, such as Ubuntu 18.04.
  
* Honor osquery RPC cancellation without breaking later result resolution. [#53774](https://github.com/elastic/beats/pull/53774) [#53645](https://github.com/elastic/beats/issues/53645)
* Restart osqueryd after runtime exit 78 while preserving startup failures. [#53776](https://github.com/elastic/beats/pull/53776) [#53645](https://github.com/elastic/beats/issues/53645)

**Packetbeat**

* Fix contact fields handling to use Contact instead of To header. [#53418](https://github.com/elastic/beats/pull/53418) 
* Fix nil map panic and missed completion for zero-body messages in AMQP parser. [#53491](https://github.com/elastic/beats/pull/53491) 
* Fix pgsql queries being paired with the wrong responses after a NULL column. [#53666](https://github.com/elastic/beats/pull/53666) 

  The pgsql parser read DataRow column lengths as unsigned, so the -1 that the PostgreSQL wire protocol uses to mark a NULL column was rejected as an invalid length. The response was dropped but its query stayed queued, so for the rest of that connection&#39;s life every published event carried the previous query&#39;s text against a later query&#39;s rows, and the last query was never published at all. A DataRow truncated part-way through a column length could also panic packetbeat.
  
* Reject invalid negative RESP array lengths in the Redis parser. [#53843](https://github.com/elastic/beats/pull/53843) 
* Fix length and count decoding in the Packetbeat Cassandra parser. [#54067](https://github.com/elastic/beats/pull/54067) 

