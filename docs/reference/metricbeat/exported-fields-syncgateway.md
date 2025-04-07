---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-syncgateway.html
---

# SyncGateway fields [exported-fields-syncgateway]

SyncGateway metrics


## syncgateway [_syncgateway]

`syncgateway` contains the information and statistics from SyncGateway.


## syncgateway [_syncgateway_2]

Couchbase Sync Gateway metrics.

**`syncgateway.syncgateway.name`**
:   Name of the database on when field `couchbase.syncgateway.type` is `db_stats`.

type: keyword



## metrics [_metrics_10]

Metrics of all databases contained in the config file of the SyncGateway instance.

**`syncgateway.syncgateway.metrics.docs.writes.conflict.count`**
:   type: long


**`syncgateway.syncgateway.metrics.docs.writes.count`**
:   type: long


**`syncgateway.syncgateway.metrics.docs.writes.bytes`**
:   type: long


**`syncgateway.syncgateway.metrics.replications.active`**
:   Number of active replications

type: long


**`syncgateway.syncgateway.metrics.replications.total`**
:   Total number of replications (active or not)

type: long


**`syncgateway.syncgateway.gsi.views.tombstones.query.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.tombstones.query.time`**
:   type: double


**`syncgateway.syncgateway.gsi.views.tombstones.query.error.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.access.query.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.access.query.error.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.access.query.time`**
:   type: double


**`syncgateway.syncgateway.gsi.views.channels.query.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.channels.query.error.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.channels.query.time`**
:   type: double


**`syncgateway.syncgateway.gsi.views.channels.star.query.time`**
:   type: double


**`syncgateway.syncgateway.gsi.views.channels.star.query.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.channels.star.query.error.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.role_access.query.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.role_access.query.error.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.role_access.query.time`**
:   type: double


**`syncgateway.syncgateway.gsi.views.sequences.query.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.sequences.query.error.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.sequences.query.time`**
:   type: double


**`syncgateway.syncgateway.gsi.views.all_docs.query.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.all_docs.query.error.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.all_docs.query.time`**
:   type: double


**`syncgateway.syncgateway.gsi.views.principals.query.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.principals.query.error.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.principals.query.time`**
:   type: double


**`syncgateway.syncgateway.gsi.views.resync.query.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.resync.query.error.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.resync.query.time`**
:   type: double


**`syncgateway.syncgateway.gsi.views.sessions.query.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.sessions.query.error.count`**
:   type: double


**`syncgateway.syncgateway.gsi.views.sessions.query.time`**
:   type: double


**`syncgateway.syncgateway.security.access_errors.count`**
:   type: double


**`syncgateway.syncgateway.security.auth.failed.count`**
:   type: double


**`syncgateway.syncgateway.security.docs_rejected.count`**
:   type: double


**`syncgateway.syncgateway.cache.channel.revs.active`**
:   type: double


**`syncgateway.syncgateway.cache.channel.revs.removal`**
:   type: double


**`syncgateway.syncgateway.cache.channel.revs.tombstone`**
:   type: double


**`syncgateway.syncgateway.cache.channel.hits`**
:   type: double


**`syncgateway.syncgateway.cache.channel.misses`**
:   type: double


**`syncgateway.syncgateway.cache.revs.hits`**
:   type: double


**`syncgateway.syncgateway.cache.revs.misses`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.pull.caught_up`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.pull.since_zero`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.pull.total.continuous`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.pull.total.one_shot`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.pull.active.continuous`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.pull.active.count`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.pull.active.one_shot`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.pull.attachment.bytes`**
:   type: long


**`syncgateway.syncgateway.cbl.replication.pull.attachment.count`**
:   type: long


**`syncgateway.syncgateway.cbl.replication.pull.request_changes.count`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.pull.request_changes.time`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.pull.rev.processing_time`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.pull.rev.send.count`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.pull.rev.send.latency`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.push.attachment.bytes`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.push.attachment.count`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.push.doc_push_count`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.push.propose_change.count`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.push.propose_change.time`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.push.sync_function.count`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.push.sync_function.time`**
:   type: double


**`syncgateway.syncgateway.cbl.replication.push.write_processing_time`**
:   type: double



## memstats [_memstats]

Dumps a large amount of information about the memory heap and garbage collector

**`syncgateway.syncgateway.memstats.BuckHashSys`**
:   type: double


**`syncgateway.syncgateway.memstats.Mallocs`**
:   type: double


**`syncgateway.syncgateway.memstats.PauseTotalNs`**
:   type: double


**`syncgateway.syncgateway.memstats.TotalAlloc`**
:   type: double


**`syncgateway.syncgateway.memstats.Alloc`**
:   type: double


**`syncgateway.syncgateway.memstats.GCSys`**
:   type: double


**`syncgateway.syncgateway.memstats.LastGC`**
:   type: double


**`syncgateway.syncgateway.memstats.MSpanSys`**
:   type: double


**`syncgateway.syncgateway.memstats.GCCPUFraction`**
:   type: double


**`syncgateway.syncgateway.memstats.HeapReleased`**
:   type: double


**`syncgateway.syncgateway.memstats.HeapSys`**
:   type: double


**`syncgateway.syncgateway.memstats.DebugGC`**
:   type: long


**`syncgateway.syncgateway.memstats.HeapIdle`**
:   type: double


**`syncgateway.syncgateway.memstats.Lookups`**
:   type: double


**`syncgateway.syncgateway.memstats.HeapObjects`**
:   type: double


**`syncgateway.syncgateway.memstats.MSpanInuse`**
:   type: double


**`syncgateway.syncgateway.memstats.NumForcedGC`**
:   type: double


**`syncgateway.syncgateway.memstats.OtherSys`**
:   type: double


**`syncgateway.syncgateway.memstats.Frees`**
:   type: double


**`syncgateway.syncgateway.memstats.NextGC`**
:   type: double


**`syncgateway.syncgateway.memstats.StackInuse`**
:   type: double


**`syncgateway.syncgateway.memstats.Sys`**
:   type: double


**`syncgateway.syncgateway.memstats.NumGC`**
:   type: double


**`syncgateway.syncgateway.memstats.EnableGC`**
:   type: long


**`syncgateway.syncgateway.memstats.HeapAlloc`**
:   type: double


**`syncgateway.syncgateway.memstats.MCacheInuse`**
:   type: double


**`syncgateway.syncgateway.memstats.MCacheSys`**
:   type: double


**`syncgateway.syncgateway.memstats.HeapInuse`**
:   type: double


**`syncgateway.syncgateway.memstats.StackSys`**
:   type: double



## memory [_memory_11]

SyncGateway memory metrics. It dumps a large amount of information about the memory heap and garbage collector

**`syncgateway.memory.BuckHashSys`**
:   type: double


**`syncgateway.memory.Mallocs`**
:   type: double


**`syncgateway.memory.PauseTotalNs`**
:   type: double


**`syncgateway.memory.TotalAlloc`**
:   type: double


**`syncgateway.memory.Alloc`**
:   type: double


**`syncgateway.memory.GCSys`**
:   type: double


**`syncgateway.memory.LastGC`**
:   type: double


**`syncgateway.memory.MSpanSys`**
:   type: double


**`syncgateway.memory.GCCPUFraction`**
:   type: double


**`syncgateway.memory.HeapReleased`**
:   type: double


**`syncgateway.memory.HeapSys`**
:   type: double


**`syncgateway.memory.DebugGC`**
:   type: long


**`syncgateway.memory.HeapIdle`**
:   type: double


**`syncgateway.memory.Lookups`**
:   type: double


**`syncgateway.memory.HeapObjects`**
:   type: double


**`syncgateway.memory.MSpanInuse`**
:   type: double


**`syncgateway.memory.NumForcedGC`**
:   type: double


**`syncgateway.memory.OtherSys`**
:   type: double


**`syncgateway.memory.Frees`**
:   type: double


**`syncgateway.memory.NextGC`**
:   type: double


**`syncgateway.memory.StackInuse`**
:   type: double


**`syncgateway.memory.Sys`**
:   type: double


**`syncgateway.memory.NumGC`**
:   type: double


**`syncgateway.memory.EnableGC`**
:   type: long


**`syncgateway.memory.HeapAlloc`**
:   type: double


**`syncgateway.memory.MCacheInuse`**
:   type: double


**`syncgateway.memory.MCacheSys`**
:   type: double


**`syncgateway.memory.HeapInuse`**
:   type: double


**`syncgateway.memory.StackSys`**
:   type: double



## replication [_replication_3]

SyncGateway per replication metrics.


## metrics [_metrics_11]

Metrics related with data replication.

**`syncgateway.replication.metrics.attachment.transferred.bytes`**
:   Number of attachment bytes transferred for this replica.

type: long


**`syncgateway.replication.metrics.attachment.transferred.count`**
:   The total number of attachments transferred since replication started.

type: long


**`syncgateway.replication.metrics.docs.checked_sent`**
:   The total number of documents checked for changes since replication started.

type: double


**`syncgateway.replication.metrics.docs.pushed.count`**
:   The total number of documents checked for changes since replication started.

type: long


**`syncgateway.replication.metrics.docs.pushed.failed`**
:   The total number of documents that failed to be pushed since replication started.

type: long


**`syncgateway.replication.id`**
:   ID of the replica.

type: keyword



## resources [_resources]

SyncGateway global resource utilization

**`syncgateway.resources.error_count`**
:   type: long


**`syncgateway.resources.goroutines_high_watermark`**
:   type: long


**`syncgateway.resources.num_goroutines`**
:   type: long


**`syncgateway.resources.process.cpu_percent_utilization`**
:   type: long


**`syncgateway.resources.process.memory_resident`**
:   type: long


**`syncgateway.resources.pub_net.recv.bytes`**
:   type: long


**`syncgateway.resources.pub_net.sent.bytes`**
:   type: long


**`syncgateway.resources.admin_net_bytes.recv`**
:   type: long


**`syncgateway.resources.admin_net_bytes.sent`**
:   type: long


**`syncgateway.resources.go_memstats.heap.alloc`**
:   type: long


**`syncgateway.resources.go_memstats.heap.idle`**
:   type: long


**`syncgateway.resources.go_memstats.heap.inuse`**
:   type: long


**`syncgateway.resources.go_memstats.heap.released`**
:   type: long


**`syncgateway.resources.go_memstats.pause.ns`**
:   type: long


**`syncgateway.resources.go_memstats.stack.inuse`**
:   type: long


**`syncgateway.resources.go_memstats.stack.sys`**
:   type: long


**`syncgateway.resources.go_memstats.sys`**
:   type: long


**`syncgateway.resources.system_memory_total`**
:   type: long


**`syncgateway.resources.warn_count`**
:   type: long


