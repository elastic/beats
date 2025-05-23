---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-mongodb.html
---

# MongoDB fields [exported-fields-mongodb]

Metrics collected from MongoDB servers.


## mongodb [_mongodb]

MongoDB metrics.


## collstats [_collstats]

MongoDB collection statistics metrics.

**`mongodb.collstats.db`**
:   Database name.

type: keyword


**`mongodb.collstats.collection`**
:   Collection name.

type: keyword


**`mongodb.collstats.name`**
:   Combination of database and collection name.

type: keyword


**`mongodb.collstats.total.time.us`**
:   Total waiting time for locks in microseconds.

type: long


**`mongodb.collstats.total.count`**
:   Total number of lock wait events.

type: long


**`mongodb.collstats.lock.read.time.us`**
:   Time waiting for read locks in microseconds.

type: long


**`mongodb.collstats.lock.read.count`**
:   Number of read lock wait events.

type: long


**`mongodb.collstats.lock.write.time.us`**
:   Time waiting for write locks in microseconds.

type: long


**`mongodb.collstats.lock.write.count`**
:   Number of write lock wait events.

type: long


**`mongodb.collstats.queries.time.us`**
:   Time running queries in microseconds.

type: long


**`mongodb.collstats.queries.count`**
:   Number of queries executed.

type: long


**`mongodb.collstats.getmore.time.us`**
:   Time asking for more cursor rows in microseconds.

type: long


**`mongodb.collstats.getmore.count`**
:   Number of times a cursor asked for more data.

type: long


**`mongodb.collstats.insert.time.us`**
:   Time inserting new documents in microseconds.

type: long


**`mongodb.collstats.insert.count`**
:   Number of document insert events.

type: long


**`mongodb.collstats.update.time.us`**
:   Time updating documents in microseconds.

type: long


**`mongodb.collstats.update.count`**
:   Number of document update events.

type: long


**`mongodb.collstats.remove.time.us`**
:   Time deleting documents in microseconds.

type: long


**`mongodb.collstats.remove.count`**
:   Number of document delete events.

type: long


**`mongodb.collstats.commands.time.us`**
:   Time executing database commands in microseconds.

type: long


**`mongodb.collstats.commands.count`**
:   Number of database commands executed.

type: long


**`mongodb.collstats.stats.stats.size`**
:   The total uncompressed size in memory of all records in a collection.

type: long


**`mongodb.collstats.stats.stats.count`**
:   The number of objects or documents in this collection.

type: long


**`mongodb.collstats.stats.stats.avgObjSize`**
:   The average size of an object in the collection (in bytes).

type: long


**`mongodb.collstats.stats.stats.storageSize`**
:   The total amount of storage allocated to this collection for document storage (in bytes).

type: long


**`mongodb.collstats.stats.stats.totalIndexSize`**
:   The total size of all indexes (in bytes).

type: long


**`mongodb.collstats.stats.stats.totalSize`**
:   The sum of the storageSize and totalIndexSize (in bytes).

type: long


**`mongodb.collstats.stats.stats.max`**
:   Shows the maximum number of documents that may be present in a capped collection.

type: long


**`mongodb.collstats.stats.stats.nindexes`**
:   The number of indexes on the collection. All collections have at least one index on the _id field.

type: long



## dbstats [_dbstats]

dbstats provides an overview of a particular mongo database. This document is most concerned with data volumes of a database.

**`mongodb.dbstats.avg_obj_size.bytes`**
:   type: long

format: bytes


**`mongodb.dbstats.collections`**
:   type: integer


**`mongodb.dbstats.data_size.bytes`**
:   type: long

format: bytes


**`mongodb.dbstats.db`**
:   type: keyword


**`mongodb.dbstats.file_size.bytes`**
:   type: long

format: bytes


**`mongodb.dbstats.index_size.bytes`**
:   type: long

format: bytes


**`mongodb.dbstats.indexes`**
:   type: long


**`mongodb.dbstats.num_extents`**
:   type: long


**`mongodb.dbstats.objects`**
:   type: long


**`mongodb.dbstats.storage_size.bytes`**
:   type: long

format: bytes


**`mongodb.dbstats.ns_size_mb.mb`**
:   type: long


**`mongodb.dbstats.data_file_version.major`**
:   type: long


**`mongodb.dbstats.data_file_version.minor`**
:   type: long


**`mongodb.dbstats.extent_free_list.num`**
:   type: long


**`mongodb.dbstats.extent_free_list.size.bytes`**
:   type: long

format: bytes



## metrics [_metrics_9]

Statistics that reflect the current use and state of a running `mongod` instance for more information, take a look at [https://docs.mongodb.com/manual/reference/command/serverStatus/#serverstatus.metrics](https://docs.mongodb.com/manual/reference/command/serverStatus/#serverstatus.metrics)


## commands [_commands]

Reports on the use of database commands. The fields in metrics.commands are the names of database commands and each value is a document that reports the total number of commands executed as well as the number of failed executions. metrics.commands.<command>.failed shows the number of times <command> failed on this mongod. metrics.commands.<command>.total shows the number of times <command> executed on this mongod.

**`mongodb.metrics.commands.is_self.failed`**
:   type: long


**`mongodb.metrics.commands.is_self.total`**
:   type: long


**`mongodb.metrics.commands.aggregate.failed`**
:   type: long


**`mongodb.metrics.commands.aggregate.total`**
:   type: long


**`mongodb.metrics.commands.build_info.failed`**
:   type: long


**`mongodb.metrics.commands.build_info.total`**
:   type: long


**`mongodb.metrics.commands.coll_stats.failed`**
:   type: long


**`mongodb.metrics.commands.coll_stats.total`**
:   type: long


**`mongodb.metrics.commands.connection_pool_stats.failed`**
:   type: long


**`mongodb.metrics.commands.connection_pool_stats.total`**
:   type: long


**`mongodb.metrics.commands.count.failed`**
:   type: long


**`mongodb.metrics.commands.count.total`**
:   type: long


**`mongodb.metrics.commands.db_stats.failed`**
:   type: long


**`mongodb.metrics.commands.db_stats.total`**
:   type: long


**`mongodb.metrics.commands.distinct.failed`**
:   type: long


**`mongodb.metrics.commands.distinct.total`**
:   type: long


**`mongodb.metrics.commands.find.failed`**
:   type: long


**`mongodb.metrics.commands.find.total`**
:   type: long


**`mongodb.metrics.commands.get_cmd_line_opts.failed`**
:   type: long


**`mongodb.metrics.commands.get_cmd_line_opts.total`**
:   type: long


**`mongodb.metrics.commands.get_last_error.failed`**
:   type: long


**`mongodb.metrics.commands.get_last_error.total`**
:   type: long


**`mongodb.metrics.commands.get_log.failed`**
:   type: long


**`mongodb.metrics.commands.get_log.total`**
:   type: long


**`mongodb.metrics.commands.get_more.failed`**
:   type: long


**`mongodb.metrics.commands.get_more.total`**
:   type: long


**`mongodb.metrics.commands.get_parameter.failed`**
:   type: long


**`mongodb.metrics.commands.get_parameter.total`**
:   type: long


**`mongodb.metrics.commands.host_info.failed`**
:   type: long


**`mongodb.metrics.commands.host_info.total`**
:   type: long


**`mongodb.metrics.commands.insert.failed`**
:   type: long


**`mongodb.metrics.commands.insert.total`**
:   type: long


**`mongodb.metrics.commands.is_master.failed`**
:   type: long


**`mongodb.metrics.commands.is_master.total`**
:   type: long


**`mongodb.metrics.commands.last_collections.failed`**
:   type: long


**`mongodb.metrics.commands.last_collections.total`**
:   type: long


**`mongodb.metrics.commands.last_commands.failed`**
:   type: long


**`mongodb.metrics.commands.last_commands.total`**
:   type: long


**`mongodb.metrics.commands.list_databased.failed`**
:   type: long


**`mongodb.metrics.commands.list_databased.total`**
:   type: long


**`mongodb.metrics.commands.list_indexes.failed`**
:   type: long


**`mongodb.metrics.commands.list_indexes.total`**
:   type: long


**`mongodb.metrics.commands.ping.failed`**
:   type: long


**`mongodb.metrics.commands.ping.total`**
:   type: long


**`mongodb.metrics.commands.profile.failed`**
:   type: long


**`mongodb.metrics.commands.profile.total`**
:   type: long


**`mongodb.metrics.commands.replset_get_rbid.failed`**
:   type: long


**`mongodb.metrics.commands.replset_get_rbid.total`**
:   type: long


**`mongodb.metrics.commands.replset_get_status.failed`**
:   type: long


**`mongodb.metrics.commands.replset_get_status.total`**
:   type: long


**`mongodb.metrics.commands.replset_heartbeat.failed`**
:   type: long


**`mongodb.metrics.commands.replset_heartbeat.total`**
:   type: long


**`mongodb.metrics.commands.replset_update_position.failed`**
:   type: long


**`mongodb.metrics.commands.replset_update_position.total`**
:   type: long


**`mongodb.metrics.commands.server_status.failed`**
:   type: long


**`mongodb.metrics.commands.server_status.total`**
:   type: long


**`mongodb.metrics.commands.update.failed`**
:   type: long


**`mongodb.metrics.commands.update.total`**
:   type: long


**`mongodb.metrics.commands.whatsmyuri.failed`**
:   type: long


**`mongodb.metrics.commands.whatsmyuri.total`**
:   type: long



## cursor [_cursor]

Contains data regarding cursor state and use.

**`mongodb.metrics.cursor.timed_out`**
:   The total number of cursors that have timed out since the server process started.

type: long



## open [_open]

Contains data regarding open cursors.

**`mongodb.metrics.cursor.open.no_timeout`**
:   The number of open cursors with the option DBQuery.Option.noTimeout set to prevent timeout.

type: long


**`mongodb.metrics.cursor.open.pinned`**
:   The number of `pinned` open cursors.

type: long


**`mongodb.metrics.cursor.open.total`**
:   The number of cursors that MongoDB is maintaining for clients.

type: long



## document [_document]

Reflects document access and modification patterns.

**`mongodb.metrics.document.deleted`**
:   The total number of documents deleted.

type: long


**`mongodb.metrics.document.inserted`**
:   The total number of documents inserted.

type: long


**`mongodb.metrics.document.returned`**
:   The total number of documents returned by queries.

type: long


**`mongodb.metrics.document.updated`**
:   The total number of documents updated.

type: long



## get_last_error [_get_last_error]

Returns the error status of the preceding write operation on the current connection.

**`mongodb.metrics.get_last_error.write_wait.ms`**
:   The total amount of time in milliseconds that the mongod has spent performing getLastError operations with write concern (i.e. w) greater than 1.

type: long


**`mongodb.metrics.get_last_error.write_wait.count`**
:   The total number of getLastError operations with a specified write concern (i.e. w) greater than 1.

type: long


**`mongodb.metrics.get_last_error.write_timeouts`**
:   The number of times that write concern operations have timed out as a result of the wtimeout threshold to getLastError.

type: long



## operation [_operation]

Holds counters for several types of update and query operations that MongoDB handles using special operation types.

**`mongodb.metrics.operation.scan_and_order`**
:   The total number of queries that return sorted numbers that cannot perform the sort operation using an index.

type: long


**`mongodb.metrics.operation.write_conflicts`**
:   The total number of queries that encountered write conflicts.

type: long



## query_executor [_query_executor]

Reports data from the query execution system.

**`mongodb.metrics.query_executor.scanned_indexes.count`**
:   The total number of index items scanned during queries and query-plan evaluation.

type: long


**`mongodb.metrics.query_executor.scanned_documents.count`**
:   The total number of documents scanned during queries and query-plan evaluation.

type: long



## replication [_replication]

Reports metrics related to the replication process. metrics.replication appears on all mongod instances, even those that aren’t members of replica sets.


## executor [_executor]

Reports on various statistics for the replication executor.

**`mongodb.metrics.replication.executor.counters.event_created`**
:   type: long


**`mongodb.metrics.replication.executor.counters.event_wait`**
:   type: long


**`mongodb.metrics.replication.executor.counters.cancels`**
:   type: long


**`mongodb.metrics.replication.executor.counters.waits`**
:   type: long


**`mongodb.metrics.replication.executor.counters.scheduled.netcmd`**
:   type: long


**`mongodb.metrics.replication.executor.counters.scheduled.dbwork`**
:   type: long


**`mongodb.metrics.replication.executor.counters.scheduled.exclusive`**
:   type: long


**`mongodb.metrics.replication.executor.counters.scheduled.work_at`**
:   type: long


**`mongodb.metrics.replication.executor.counters.scheduled.work`**
:   type: long


**`mongodb.metrics.replication.executor.counters.scheduled.failures`**
:   type: long


**`mongodb.metrics.replication.executor.queues.in_progress.network`**
:   type: long


**`mongodb.metrics.replication.executor.queues.in_progress.dbwork`**
:   type: long


**`mongodb.metrics.replication.executor.queues.in_progress.exclusive`**
:   type: long


**`mongodb.metrics.replication.executor.queues.sleepers`**
:   type: long


**`mongodb.metrics.replication.executor.queues.ready`**
:   type: long


**`mongodb.metrics.replication.executor.queues.free`**
:   type: long


**`mongodb.metrics.replication.executor.unsignaled_events`**
:   type: long


**`mongodb.metrics.replication.executor.event_waiters`**
:   type: long


**`mongodb.metrics.replication.executor.shutting_down`**
:   type: boolean


**`mongodb.metrics.replication.executor.network_interface`**
:   type: keyword



## apply [_apply]

Reports on the application of operations from the replication oplog.

**`mongodb.metrics.replication.apply.attempts_to_become_secondary`**
:   type: long



## batches [_batches]

Reports on the oplog application process on secondaries members of replica sets.

**`mongodb.metrics.replication.apply.batches.count`**
:   The total number of batches applied across all databases.

type: long


**`mongodb.metrics.replication.apply.batches.time.ms`**
:   The total amount of time in milliseconds the mongod has spent applying operations from the oplog.

type: long


**`mongodb.metrics.replication.apply.ops`**
:   The total number of oplog operations applied.

type: long



## buffer [_buffer]

MongoDB buffers oplog operations from the replication sync source buffer before applying oplog entries in a batch. metrics.replication.buffer provides a way to track the oplog buffer.

**`mongodb.metrics.replication.buffer.count`**
:   The current number of operations in the oplog buffer.

type: long


**`mongodb.metrics.replication.buffer.max_size.bytes`**
:   The maximum size of the buffer. This value is a constant setting in the mongod, and is not configurable.

type: long


**`mongodb.metrics.replication.buffer.size.bytes`**
:   The current size of the contents of the oplog buffer.

type: long



## initial_sync [_initial_sync]

Report initial sync status

**`mongodb.metrics.replication.initial_sync.completed`**
:   type: long


**`mongodb.metrics.replication.initial_sync.failed_attempts`**
:   type: long


**`mongodb.metrics.replication.initial_sync.failures`**
:   type: long



## network [_network_8]

Reports network use by the replication process.

**`mongodb.metrics.replication.network.bytes`**
:   The total amount of data read from the replication sync source.

type: long



## getmores [_getmores]

Reports on the getmore operations, which are requests for additional results from the oplog cursor as part of the oplog replication process.

**`mongodb.metrics.replication.network.getmores.count`**
:   The total number of getmore operations

type: long


**`mongodb.metrics.replication.network.getmores.time.ms`**
:   The total amount of time required to collect data from getmore operations.

type: long


**`mongodb.metrics.replication.network.ops`**
:   The total number of operations read from the replication source.

type: long


**`mongodb.metrics.replication.network.reders_created`**
:   The total number of oplog query processes created.

type: long



## preload [_preload]

Reports on the `pre-fetch` stage, where MongoDB loads documents and indexes into RAM to improve replication throughput.


## docs [_docs]

Reports on the documents loaded into memory during the pre-fetch stage.

**`mongodb.metrics.replication.preload.docs.count`**
:   The total number of documents loaded during the pre-fetch stage of replication.

type: long


**`mongodb.metrics.replication.preload.docs.time.ms`**
:   type: long



## indexes [_indexes]

Reports on the index items loaded into memory during the pre-fetch stage of replication.

**`mongodb.metrics.replication.preload.indexes.count`**
:   The total number of index entries loaded by members before updating documents as part of the pre-fetch stage of replication.

type: long


**`mongodb.metrics.replication.preload.indexes.time.ms`**
:   The total amount of time, in milliseconds, spent loading index entries as part of the pre-fetch stage of replication.

type: long


**`mongodb.metrics.storage.free_list.search.bucket_exhausted`**
:   The number of times that mongod has checked the free list without finding a suitably large record allocation.

type: long


**`mongodb.metrics.storage.free_list.search.requests`**
:   The number of times mongod has searched for available record allocations.

type: long


**`mongodb.metrics.storage.free_list.search.scanned`**
:   The number of available record allocations mongod has searched.

type: long



## ttl [_ttl_2]

Reports on the operation of the resource use of the ttl index process.

**`mongodb.metrics.ttl.deleted_documents.count`**
:   The total number of documents deleted from collections with a ttl index.

type: long


**`mongodb.metrics.ttl.passes.count`**
:   The number of times the background process removes documents from collections with a ttl index.

type: long



## replstatus [_replstatus]

replstatus provides an overview of replica set status.


## oplog [_oplog]

oplog provides an overview of replication oplog status, which is retrieved from db.getReplicationInfo().

**`mongodb.replstatus.oplog.size.allocated`**
:   The total amount of space used by the replstatus in bytes.

type: long

format: bytes


**`mongodb.replstatus.oplog.size.used`**
:   total amount of space allocated to the replstatus in bytes.

type: long

format: bytes


**`mongodb.replstatus.oplog.first.timestamp`**
:   Timestamp of the first (i.e. earliest) operation in the replstatus

type: long


**`mongodb.replstatus.oplog.last.timestamp`**
:   Timestamp of the last (i.e. latest) operation in the replstatus

type: long


**`mongodb.replstatus.oplog.window`**
:   The difference between the first and last operation in the replstatus.

type: long


**`mongodb.replstatus.set_name`**
:   The name of the replica set.

type: keyword


**`mongodb.replstatus.server_date`**
:   Reflects the current time according to the server that processed the replSetGetStatus command.

type: date


**`mongodb.replstatus.optimes.last_committed`**
:   Information, from the viewpoint of this member, regarding the most recent operation that has been written to a majority of replica set members.

type: long


**`mongodb.replstatus.optimes.applied`**
:   Information, from the viewpoint of this member, regarding the most recent operation that has been applied to this member of the replica set.

type: long


**`mongodb.replstatus.optimes.durable`**
:   Information, from the viewpoint of this member, regarding the most recent operation that has been written to the journal of this member of the replica set.

type: long



## lag [_lag]

Delay between a write operation on the primary and its copy to a secondary

**`mongodb.replstatus.lag.max`**
:   Difference between optime of primary and slowest secondary

type: long

format: duration


**`mongodb.replstatus.lag.min`**
:   Difference between optime of primary and fastest secondary

type: long

format: duration



## headroom [_headroom]

Difference between the primary’s oplog window and the replication lag of the secondary

**`mongodb.replstatus.headroom.max`**
:   Difference between primary’s oplog window and the replication lag of the fastest secondary

type: long

format: duration


**`mongodb.replstatus.headroom.min`**
:   Difference between primary’s oplog window and the replication lag of the slowest secondary

type: long

format: duration



## members [_members]

Provides information about members of replica set grouped by their state

**`mongodb.replstatus.members.primary.host`**
:   Host address of the primary

type: keyword


**`mongodb.replstatus.members.primary.optime`**
:   Optime of primary

type: keyword


**`mongodb.replstatus.members.secondary.hosts`**
:   List of secondary hosts

type: keyword


**`mongodb.replstatus.members.secondary.optimes`**
:   Optimes of secondaries

type: keyword


**`mongodb.replstatus.members.secondary.count`**
:   type: long


**`mongodb.replstatus.members.recovering.hosts`**
:   List of recovering members hosts

type: keyword


**`mongodb.replstatus.members.recovering.count`**
:   Count of members in the `recovering` state

type: long


**`mongodb.replstatus.members.unknown.hosts`**
:   List of members' hosts in the `unknown` state

type: keyword


**`mongodb.replstatus.members.unknown.count`**
:   Count of members with `unknown` state

type: long


**`mongodb.replstatus.members.startup2.hosts`**
:   List of initializing members hosts

type: keyword


**`mongodb.replstatus.members.startup2.count`**
:   Count of members in the `startup2` state

type: long


**`mongodb.replstatus.members.arbiter.hosts`**
:   List of arbiters hosts

type: keyword


**`mongodb.replstatus.members.arbiter.count`**
:   Count of arbiters

type: long


**`mongodb.replstatus.members.down.hosts`**
:   List of `down` members hosts

type: keyword


**`mongodb.replstatus.members.down.count`**
:   Count of `down` members

type: long


**`mongodb.replstatus.members.rollback.hosts`**
:   List of members in the `rollback` state

type: keyword


**`mongodb.replstatus.members.rollback.count`**
:   Count of members in the `rollback` state

type: long


**`mongodb.replstatus.members.unhealthy.hosts`**
:   List of members' hosts with healthy = false

type: keyword


**`mongodb.replstatus.members.unhealthy.count`**
:   Count of unhealthy members

type: long



## status [_status_6]

MongoDB server status metrics.

**`mongodb.status.version`**
:   Instance version.

type: alias

alias to: service.version


**`mongodb.status.process`**
:   The current MongoDB process. Possible values are mongos or mongod.

type: alias

alias to: process.name


**`mongodb.status.uptime.ms`**
:   Instance uptime in milliseconds.

type: long


**`mongodb.status.local_time`**
:   Local time as reported by the MongoDB instance.

type: date


**`mongodb.status.asserts.regular`**
:   Number of regular assertions produced by the server.

type: long


**`mongodb.status.asserts.warning`**
:   Number of warning assertions produced by the server.

type: long


**`mongodb.status.asserts.msg`**
:   Number of msg assertions produced by the server.

type: long


**`mongodb.status.asserts.user`**
:   Number of user assertions produced by the server.

type: long


**`mongodb.status.asserts.rollovers`**
:   Number of rollovers assertions produced by the server.

type: long



## connections [_connections_3]

Data regarding the current status of incoming connections and availability of the database server.

**`mongodb.status.connections.current`**
:   The number of connections to the database server from clients. This number includes the current shell session. Consider the value of `available` to add more context to this datum.

type: long


**`mongodb.status.connections.available`**
:   The number of unused available incoming connections the database can provide.

type: long


**`mongodb.status.connections.total_created`**
:   A count of all incoming connections created to the server. This number includes connections that have since closed.

type: long



## extra_info [_extra_info]

Platform specific data.

**`mongodb.status.extra_info.heap_usage.bytes`**
:   The total size in bytes of heap space used by the database process. Only available on Unix/Linux.

type: long

format: bytes


**`mongodb.status.extra_info.page_faults`**
:   The total number of page faults that require disk operations. Page faults refer to operations that require the database server to access data that isn’t available in active memory.

type: long



## global_lock [_global_lock]

Reports on lock state of the database.

**`mongodb.status.global_lock.total_time.us`**
:   The time, in microseconds, since the database last started and created the globalLock. This is roughly equivalent to total server uptime.

type: long



## current_queue [_current_queue]

The number of operations queued because of a lock.

**`mongodb.status.global_lock.current_queue.total`**
:   The total number of operations queued waiting for the lock (i.e., the sum of current_queue.readers and current_queue.writers).

type: long


**`mongodb.status.global_lock.current_queue.readers`**
:   The number of operations that are currently queued and waiting for the read lock.

type: long


**`mongodb.status.global_lock.current_queue.writers`**
:   The number of operations that are currently queued and waiting for the write lock.

type: long



## active_clients [_active_clients]

The number of connected clients and the read and write operations performed by these clients.

**`mongodb.status.global_lock.active_clients.total`**
:   Total number of the active client connections performing read or write operations.

type: long


**`mongodb.status.global_lock.active_clients.readers`**
:   The number of the active client connections performing read operations.

type: long


**`mongodb.status.global_lock.active_clients.writers`**
:   The number of the active client connections performing write operations.

type: long



## locks [_locks]

A document that reports for each lock <type>, data on lock <mode>s. The possible lock <type>s are global, database, collection, metadata and oplog. The possible <mode>s are r, w, R and W which respresent shared, exclusive, intent shared and intent exclusive. locks.<type>.acquire.count.<mode> shows the number of times the lock was acquired in the specified mode. locks.<type>.wait.count.<mode> shows the number of times the locks.acquireCount lock acquisitions encountered waits because the locks were held in a conflicting mode. locks.<type>.wait.us.<mode> shows the cumulative wait time in microseconds for the lock acquisitions. locks.<type>.deadlock.count.<mode> shows the number of times the lock acquisitions encountered deadlocks.

**`mongodb.status.locks.global.acquire.count.r`**
:   type: long


**`mongodb.status.locks.global.acquire.count.w`**
:   type: long


**`mongodb.status.locks.global.acquire.count.R`**
:   type: long


**`mongodb.status.locks.global.acquire.count.W`**
:   type: long


**`mongodb.status.locks.global.wait.count.r`**
:   type: long


**`mongodb.status.locks.global.wait.count.w`**
:   type: long


**`mongodb.status.locks.global.wait.count.R`**
:   type: long


**`mongodb.status.locks.global.wait.count.W`**
:   type: long


**`mongodb.status.locks.global.wait.us.r`**
:   type: long


**`mongodb.status.locks.global.wait.us.w`**
:   type: long


**`mongodb.status.locks.global.wait.us.R`**
:   type: long


**`mongodb.status.locks.global.wait.us.W`**
:   type: long


**`mongodb.status.locks.global.deadlock.count.r`**
:   type: long


**`mongodb.status.locks.global.deadlock.count.w`**
:   type: long


**`mongodb.status.locks.global.deadlock.count.R`**
:   type: long


**`mongodb.status.locks.global.deadlock.count.W`**
:   type: long


**`mongodb.status.locks.database.acquire.count.r`**
:   type: long


**`mongodb.status.locks.database.acquire.count.w`**
:   type: long


**`mongodb.status.locks.database.acquire.count.R`**
:   type: long


**`mongodb.status.locks.database.acquire.count.W`**
:   type: long


**`mongodb.status.locks.database.wait.count.r`**
:   type: long


**`mongodb.status.locks.database.wait.count.w`**
:   type: long


**`mongodb.status.locks.database.wait.count.R`**
:   type: long


**`mongodb.status.locks.database.wait.count.W`**
:   type: long


**`mongodb.status.locks.database.wait.us.r`**
:   type: long


**`mongodb.status.locks.database.wait.us.w`**
:   type: long


**`mongodb.status.locks.database.wait.us.R`**
:   type: long


**`mongodb.status.locks.database.wait.us.W`**
:   type: long


**`mongodb.status.locks.database.deadlock.count.r`**
:   type: long


**`mongodb.status.locks.database.deadlock.count.w`**
:   type: long


**`mongodb.status.locks.database.deadlock.count.R`**
:   type: long


**`mongodb.status.locks.database.deadlock.count.W`**
:   type: long


**`mongodb.status.locks.collection.acquire.count.r`**
:   type: long


**`mongodb.status.locks.collection.acquire.count.w`**
:   type: long


**`mongodb.status.locks.collection.acquire.count.R`**
:   type: long


**`mongodb.status.locks.collection.acquire.count.W`**
:   type: long


**`mongodb.status.locks.collection.wait.count.r`**
:   type: long


**`mongodb.status.locks.collection.wait.count.w`**
:   type: long


**`mongodb.status.locks.collection.wait.count.R`**
:   type: long


**`mongodb.status.locks.collection.wait.count.W`**
:   type: long


**`mongodb.status.locks.collection.wait.us.r`**
:   type: long


**`mongodb.status.locks.collection.wait.us.w`**
:   type: long


**`mongodb.status.locks.collection.wait.us.R`**
:   type: long


**`mongodb.status.locks.collection.wait.us.W`**
:   type: long


**`mongodb.status.locks.collection.deadlock.count.r`**
:   type: long


**`mongodb.status.locks.collection.deadlock.count.w`**
:   type: long


**`mongodb.status.locks.collection.deadlock.count.R`**
:   type: long


**`mongodb.status.locks.collection.deadlock.count.W`**
:   type: long


**`mongodb.status.locks.meta_data.acquire.count.r`**
:   type: long


**`mongodb.status.locks.meta_data.acquire.count.w`**
:   type: long


**`mongodb.status.locks.meta_data.acquire.count.R`**
:   type: long


**`mongodb.status.locks.meta_data.acquire.count.W`**
:   type: long


**`mongodb.status.locks.meta_data.wait.count.r`**
:   type: long


**`mongodb.status.locks.meta_data.wait.count.w`**
:   type: long


**`mongodb.status.locks.meta_data.wait.count.R`**
:   type: long


**`mongodb.status.locks.meta_data.wait.count.W`**
:   type: long


**`mongodb.status.locks.meta_data.wait.us.r`**
:   type: long


**`mongodb.status.locks.meta_data.wait.us.w`**
:   type: long


**`mongodb.status.locks.meta_data.wait.us.R`**
:   type: long


**`mongodb.status.locks.meta_data.wait.us.W`**
:   type: long


**`mongodb.status.locks.meta_data.deadlock.count.r`**
:   type: long


**`mongodb.status.locks.meta_data.deadlock.count.w`**
:   type: long


**`mongodb.status.locks.meta_data.deadlock.count.R`**
:   type: long


**`mongodb.status.locks.meta_data.deadlock.count.W`**
:   type: long


**`mongodb.status.locks.oplog.acquire.count.r`**
:   type: long


**`mongodb.status.locks.oplog.acquire.count.w`**
:   type: long


**`mongodb.status.locks.oplog.acquire.count.R`**
:   type: long


**`mongodb.status.locks.oplog.acquire.count.W`**
:   type: long


**`mongodb.status.locks.oplog.wait.count.r`**
:   type: long


**`mongodb.status.locks.oplog.wait.count.w`**
:   type: long


**`mongodb.status.locks.oplog.wait.count.R`**
:   type: long


**`mongodb.status.locks.oplog.wait.count.W`**
:   type: long


**`mongodb.status.locks.oplog.wait.us.r`**
:   type: long


**`mongodb.status.locks.oplog.wait.us.w`**
:   type: long


**`mongodb.status.locks.oplog.wait.us.R`**
:   type: long


**`mongodb.status.locks.oplog.wait.us.W`**
:   type: long


**`mongodb.status.locks.oplog.deadlock.count.r`**
:   type: long


**`mongodb.status.locks.oplog.deadlock.count.w`**
:   type: long


**`mongodb.status.locks.oplog.deadlock.count.R`**
:   type: long


**`mongodb.status.locks.oplog.deadlock.count.W`**
:   type: long



## network [_network_9]

Platform specific data.

**`mongodb.status.network.in.bytes`**
:   The amount of network traffic, in bytes, received by this database.

type: long

format: bytes


**`mongodb.status.network.out.bytes`**
:   The amount of network traffic, in bytes, sent from this database.

type: long

format: bytes


**`mongodb.status.network.requests`**
:   The total number of requests received by the server.

type: long



## ops.latencies [_ops_latencies]

Operation latencies for the database as a whole. Only mongod instances report this metric.

**`mongodb.status.ops.latencies.reads.latency`**
:   Total combined latency in microseconds.

type: long


**`mongodb.status.ops.latencies.reads.count`**
:   Total number of read operations performed on the collection since startup.

type: long


**`mongodb.status.ops.latencies.writes.latency`**
:   Total combined latency in microseconds.

type: long


**`mongodb.status.ops.latencies.writes.count`**
:   Total number of write operations performed on the collection since startup.

type: long


**`mongodb.status.ops.latencies.commands.latency`**
:   Total combined latency in microseconds.

type: long


**`mongodb.status.ops.latencies.commands.count`**
:   Total number of commands performed on the collection since startup.

type: long



## ops.counters [_ops_counters]

An overview of database operations by type.

**`mongodb.status.ops.counters.insert`**
:   The total number of insert operations received since the mongod instance last started.

type: long


**`mongodb.status.ops.counters.query`**
:   The total number of queries received since the mongod instance last started.

type: long


**`mongodb.status.ops.counters.update`**
:   The total number of update operations received since the mongod instance last started.

type: long


**`mongodb.status.ops.counters.delete`**
:   The total number of delete operations received since the mongod instance last started.

type: long


**`mongodb.status.ops.counters.getmore`**
:   The total number of getmore operations received since the mongod instance last started.

type: long


**`mongodb.status.ops.counters.command`**
:   The total number of commands issued to the database since the mongod instance last started.

type: long



## ops.replicated [_ops_replicated]

An overview of database replication operations by type.

**`mongodb.status.ops.replicated.insert`**
:   The total number of replicated insert operations received since the mongod instance last started.

type: long


**`mongodb.status.ops.replicated.query`**
:   The total number of replicated queries received since the mongod instance last started.

type: long


**`mongodb.status.ops.replicated.update`**
:   The total number of replicated update operations received since the mongod instance last started.

type: long


**`mongodb.status.ops.replicated.delete`**
:   The total number of replicated delete operations received since the mongod instance last started.

type: long


**`mongodb.status.ops.replicated.getmore`**
:   The total number of replicated getmore operations received since the mongod instance last started.

type: long


**`mongodb.status.ops.replicated.command`**
:   The total number of replicated commands issued to the database since the mongod instance last started.

type: long



## memory [_memory_9]

Data about the current memory usage of the mongod server.

**`mongodb.status.memory.bits`**
:   Either 64 or 32, depending on which target architecture was specified during the mongod compilation process.

type: long


**`mongodb.status.memory.resident.mb`**
:   The amount of RAM, in megabytes (MB), currently used by the database process.

type: long


**`mongodb.status.memory.virtual.mb`**
:   The amount, in megabytes (MB), of virtual memory used by the mongod process.

type: long


**`mongodb.status.memory.mapped.mb`**
:   The amount of mapped memory, in megabytes (MB), used by the database. Because MongoDB uses memory-mapped files, this value is likely to be to be roughly equivalent to the total size of your database or databases.

type: long


**`mongodb.status.memory.mapped_with_journal.mb`**
:   The amount of mapped memory, in megabytes (MB), including the memory used for journaling.

type: long


**`mongodb.status.write_backs_queued`**
:   True when there are operations from a mongos instance queued for retrying.

type: boolean


**`mongodb.status.storage_engine.name`**
:   A string that represents the name of the current storage engine.

type: keyword



## wired_tiger [_wired_tiger]

Statistics about the WiredTiger storage engine.


## concurrent_transactions [_concurrent_transactions]

Statistics about the transactions currently in progress.

**`mongodb.status.wired_tiger.concurrent_transactions.write.out`**
:   Number of concurrent write transaction in progress.

type: long


**`mongodb.status.wired_tiger.concurrent_transactions.write.available`**
:   Number of concurrent write tickets available.

type: long


**`mongodb.status.wired_tiger.concurrent_transactions.write.total_tickets`**
:   Number of total write tickets.

type: long


**`mongodb.status.wired_tiger.concurrent_transactions.read.out`**
:   Number of concurrent read transaction in progress.

type: long


**`mongodb.status.wired_tiger.concurrent_transactions.read.available`**
:   Number of concurrent read tickets available.

type: long


**`mongodb.status.wired_tiger.concurrent_transactions.read.total_tickets`**
:   Number of total read tickets.

type: long



## cache [_cache_2]

Statistics about the cache and page evictions from the cache.

**`mongodb.status.wired_tiger.cache.maximum.bytes`**
:   Maximum cache size.

type: long

format: bytes


**`mongodb.status.wired_tiger.cache.used.bytes`**
:   Size in byte of the data currently in cache.

type: long

format: bytes


**`mongodb.status.wired_tiger.cache.dirty.bytes`**
:   Size in bytes of the dirty data in the cache.

type: long

format: bytes


**`mongodb.status.wired_tiger.cache.pages.read`**
:   Number of pages read into the cache.

type: long


**`mongodb.status.wired_tiger.cache.pages.write`**
:   Number of pages written from the cache.

type: long


**`mongodb.status.wired_tiger.cache.pages.evicted`**
:   Number of pages evicted from the cache.

type: long



## log [_log_2]

Statistics about the write ahead log used by WiredTiger.

**`mongodb.status.wired_tiger.log.size.bytes`**
:   Total log size in bytes.

type: long

format: bytes


**`mongodb.status.wired_tiger.log.write.bytes`**
:   Number of bytes written into the log.

type: long

format: bytes


**`mongodb.status.wired_tiger.log.max_file_size.bytes`**
:   Maximum file size.

type: long

format: bytes


**`mongodb.status.wired_tiger.log.flushes`**
:   Number of flush operations.

type: long


**`mongodb.status.wired_tiger.log.writes`**
:   Number of write operations.

type: long


**`mongodb.status.wired_tiger.log.scans`**
:   Number of scan operations.

type: long


**`mongodb.status.wired_tiger.log.syncs`**
:   Number of sync operations.

type: long



## background_flushing [_background_flushing]

Data about the process MongoDB uses to write data to disk. This data is only available for instances that use the MMAPv1 storage engine.

**`mongodb.status.background_flushing.flushes`**
:   A counter that collects the number of times the database has flushed all writes to disk.

type: long


**`mongodb.status.background_flushing.total.ms`**
:   The total number of milliseconds (ms) that the mongod processes have spent writing (i.e. flushing) data to disk. Because this is an absolute value, consider the value of `flushes` and `average_ms` to provide better context for this datum.

type: long


**`mongodb.status.background_flushing.average.ms`**
:   The average time spent flushing to disk per flush event.

type: long


**`mongodb.status.background_flushing.last.ms`**
:   The amount of time, in milliseconds, that the last flush operation took to complete.

type: long


**`mongodb.status.background_flushing.last_finished`**
:   A timestamp of the last completed flush operation.

type: date



## journaling [_journaling]

Data about the journaling-related operations and performance. Journaling information only appears for mongod instances that use the MMAPv1 storage engine and have journaling enabled.

**`mongodb.status.journaling.commits`**
:   The number of transactions written to the journal during the last journal group commit interval.

type: long


**`mongodb.status.journaling.journaled.mb`**
:   The amount of data in megabytes (MB) written to journal during the last journal group commit interval.

type: long


**`mongodb.status.journaling.write_to_data_files.mb`**
:   The amount of data in megabytes (MB) written from journal to the data files during the last journal group commit interval.

type: long


**`mongodb.status.journaling.compression`**
:   The compression ratio of the data written to the journal.

type: long


**`mongodb.status.journaling.commits_in_write_lock`**
:   Count of the commits that occurred while a write lock was held. Commits in a write lock indicate a MongoDB node under a heavy write load and call for further diagnosis.

type: long


**`mongodb.status.journaling.early_commits`**
:   The number of times MongoDB requested a commit before the scheduled journal group commit interval.

type: long



## times [_times]

Information about the performance of the mongod instance during the various phases of journaling in the last journal group commit interval.

**`mongodb.status.journaling.times.dt.ms`**
:   The amount of time over which MongoDB collected the times data. Use this field to provide context to the other times field values.

type: long


**`mongodb.status.journaling.times.prep_log_buffer.ms`**
:   The amount of time spent preparing to write to the journal. Smaller values indicate better journal performance.

type: long


**`mongodb.status.journaling.times.write_to_journal.ms`**
:   The amount of time spent actually writing to the journal. File system speeds and device interfaces can affect performance.

type: long


**`mongodb.status.journaling.times.write_to_data_files.ms`**
:   The amount of time spent writing to data files after journaling. File system speeds and device interfaces can affect performance.

type: long


**`mongodb.status.journaling.times.remap_private_view.ms`**
:   The amount of time spent remapping copy-on-write memory mapped views. Smaller values indicate better journal performance.

type: long


**`mongodb.status.journaling.times.commits.ms`**
:   The amount of time spent for commits.

type: long


**`mongodb.status.journaling.times.commits_in_write_lock.ms`**
:   The amount of time spent for commits that occurred while a write lock was held.

type: long


