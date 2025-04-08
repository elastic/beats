---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-mysql.html
---

# MySQL fields [exported-fields-mysql]

MySQL server status metrics collected from MySQL.


## mysql [_mysql]

`mysql` contains the metrics that were obtained from MySQL query.


## galera_status [_galera_status]

`galera_status` contains the metrics that were obtained by the status SQL query on Galera.


## apply [_apply_2]

Apply status fields.

**`mysql.galera_status.apply.oooe`**
:   How often applier started write-set applying out-of-order (parallelization efficiency).

type: double


**`mysql.galera_status.apply.oool`**
:   How often write-set was so slow to apply that write-set with higher seqno’s were applied earlier. Values closer to 0 refer to a greater gap between slow and fast write-sets.

type: double


**`mysql.galera_status.apply.window`**
:   Average distance between highest and lowest concurrently applied seqno.

type: double



## cert [_cert]

Certification status fields.

**`mysql.galera_status.cert.deps_distance`**
:   Average distance between highest and lowest seqno value that can be possibly applied in parallel (potential degree of parallelization).

type: double


**`mysql.galera_status.cert.index_size`**
:   The number of entries in the certification index.

type: long


**`mysql.galera_status.cert.interval`**
:   Average number of transactions received while a transaction replicates.

type: double



## cluster [_cluster_2]

Cluster status fields.

**`mysql.galera_status.cluster.conf_id`**
:   Total number of cluster membership changes happened.

type: long


**`mysql.galera_status.cluster.size`**
:   Current number of members in the cluster.

type: long


**`mysql.galera_status.cluster.status`**
:   Status of this cluster component. That is, whether the node is part of a PRIMARY or NON_PRIMARY component.

type: keyword



## commit [_commit_3]

Commit status fields.

**`mysql.galera_status.commit.oooe`**
:   How often a transaction was committed out of order.

type: double


**`mysql.galera_status.commit.window`**
:   Average distance between highest and lowest concurrently committed seqno.

type: long


**`mysql.galera_status.connected`**
:   If the value is OFF, the node has not yet connected to any of the cluster components. This may be due to misconfiguration. Check the error log for proper diagnostics.

type: keyword



## evs [_evs]

Evs Fields.

**`mysql.galera_status.evs.evict`**
:   Lists the UUID’s of all nodes evicted from the cluster. Evicted nodes cannot rejoin the cluster until you restart their mysqld processes.

type: keyword


**`mysql.galera_status.evs.state`**
:   Shows the internal state of the EVS Protocol.

type: keyword



## flow_ctl [_flow_ctl]

Flow Control fields.

**`mysql.galera_status.flow_ctl.paused`**
:   The fraction of time since the last FLUSH STATUS command that replication was paused due to flow control. In other words, how much the slave lag is slowing down the cluster.

type: double


**`mysql.galera_status.flow_ctl.paused_ns`**
:   The total time spent in a paused state measured in nanoseconds.

type: long


**`mysql.galera_status.flow_ctl.recv`**
:   Returns the number of FC_PAUSE events the node has received, including those the node has sent. Unlike most status variables, the counter for this one does not reset every time you run the query.

type: long


**`mysql.galera_status.flow_ctl.sent`**
:   Returns the number of FC_PAUSE events the node has sent. Unlike most status variables, the counter for this one does not reset every time you run the query.

type: long


**`mysql.galera_status.last_committed`**
:   The sequence number, or seqno, of the last committed transaction.

type: long



## local [_local]

Node specific Cluster status fields.

**`mysql.galera_status.local.bf_aborts`**
:   Total number of local transactions that were aborted by slave transactions while in execution.

type: long


**`mysql.galera_status.local.cert_failures`**
:   Total number of local transactions that failed certification test.

type: long


**`mysql.galera_status.local.commits`**
:   Total number of local transactions committed.

type: long



## recv [_recv]

Node specific recv fields.

**`mysql.galera_status.local.recv.queue`**
:   Current (instantaneous) length of the recv queue.

type: long


**`mysql.galera_status.local.recv.queue_avg`**
:   Recv queue length averaged over interval since the last FLUSH STATUS command. Values considerably larger than 0.0 mean that the node cannot apply write-sets as fast as they are received and will generate a lot of replication throttling.

type: double


**`mysql.galera_status.local.recv.queue_max`**
:   The maximum length of the recv queue since the last FLUSH STATUS command.

type: long


**`mysql.galera_status.local.recv.queue_min`**
:   The minimum length of the recv queue since the last FLUSH STATUS command.

type: long


**`mysql.galera_status.local.replays`**
:   Total number of transaction replays due to asymmetric lock granularity.

type: long



## send [_send]

Node specific sent fields.

**`mysql.galera_status.local.send.queue`**
:   Current (instantaneous) length of the send queue.

type: long


**`mysql.galera_status.local.send.queue_avg`**
:   Send queue length averaged over time since the last FLUSH STATUS command. Values considerably larger than 0.0 indicate replication throttling or network throughput issue.

type: double


**`mysql.galera_status.local.send.queue_max`**
:   The maximum length of the send queue since the last FLUSH STATUS command.

type: long


**`mysql.galera_status.local.send.queue_min`**
:   The minimum length of the send queue since the last FLUSH STATUS command.

type: long


**`mysql.galera_status.local.state`**
:   Internal Galera Cluster FSM state number.

type: keyword


**`mysql.galera_status.ready`**
:   Whether the server is ready to accept queries.

type: keyword



## received [_received]

Write-Set receive status fields.

**`mysql.galera_status.received.count`**
:   Total number of write-sets received from other nodes.

type: long


**`mysql.galera_status.received.bytes`**
:   Total size of write-sets received from other nodes.

type: long



## repl [_repl]

Replication status fields.

**`mysql.galera_status.repl.data_bytes`**
:   Total size of data replicated.

type: long


**`mysql.galera_status.repl.keys`**
:   Total number of keys replicated.

type: long


**`mysql.galera_status.repl.keys_bytes`**
:   Total size of keys replicated.

type: long


**`mysql.galera_status.repl.other_bytes`**
:   Total size of other bits replicated.

type: long


**`mysql.galera_status.repl.count`**
:   Total number of write-sets replicated (sent to other nodes).

type: long


**`mysql.galera_status.repl.bytes`**
:   Total size of write-sets replicated.

type: long



## performance [_performance_4]

`performance` contains metrics related to the performance of a MySQL instance


## events_statements [_events_statements]

Records statement events summarized by schema and digest

**`mysql.performance.events_statements.max.timer.wait`**
:   Maximum wait time of the summarized events that are timed

type: long


**`mysql.performance.events_statements.last.seen`**
:   Time at which the digest was most recently seen

type: date


**`mysql.performance.events_statements.quantile.95`**
:   The 95th percentile of the statement latency, in picoseconds

type: long


**`mysql.performance.events_statements.digest`**
:   Performance schema digest

type: text


**`mysql.performance.events_statements.count.star`**
:   Number of summarized events

type: long


**`mysql.performance.events_statements.avg.timer.wait`**
:   Average wait time of the summarized events that are timed

type: long


**`mysql.performance.events_statements.schemaname`**
:   Schema name.

type: keyword



## table_io_waits [_table_io_waits]

Records table I/O waits by index

**`mysql.performance.table_io_waits.object.schema`**
:   Schema name

type: keyword


**`mysql.performance.table_io_waits.object.name`**
:   Table name

type: keyword


**`mysql.performance.table_io_waits.index.name`**
:   Name of the index that was used when the table I/O wait event was recorded. PRIMARY indicates that table I/O used the primary index. NULL means that table I/O used no index. Inserts are counted against INDEX_NAME = NULL

type: keyword


**`mysql.performance.table_io_waits.count.fetch`**
:   Number of all fetch operations > 0

type: long



## query [_query_2]

`query` metricset fetches custom queries from the user to a MySQL instance.


## status [_status_7]

`status` contains the metrics that were obtained by the status SQL query.


## aborted [_aborted]

Aborted status fields.

**`mysql.status.aborted.clients`**
:   The number of connections that were aborted because the client died without closing the connection properly.

type: long


**`mysql.status.aborted.connects`**
:   The number of failed attempts to connect to the MySQL server.

type: long



## connection [_connection_2]


## errors [_errors]

**`mysql.status.connection.errors.peer_address`**
:   The number of errors that occurred while searching for connecting client IP addresses.

type: long


**`mysql.status.connection.errors.accept`**
:   The number of errors that occurred during calls to accept() on the listening port.

type: long


**`mysql.status.connection.errors.internal`**
:   The number of connections refused due to internal errors in the server, such as failure to start a new thread or an out-of-memory condition.

type: long


**`mysql.status.connection.errors.max`**
:   The number of connections refused because the server max_connections limit was reached. thread or an out-of-memory condition.

type: long


**`mysql.status.connection.errors.tcpwrap`**
:   The number of connections refused by the libwrap library.

type: long


**`mysql.status.connection.errors.select`**
:   The number of errors that occurred during calls to select() or poll() on the listening port. (Failure of this operation does not necessarily means a client connection was rejected.)

type: long



## cache [_cache_3]


## ssl [_ssl_8]

SSL session cache hits and misses.

**`mysql.status.cache.ssl.hits`**
:   The number of SSL session cache hits.

type: long


**`mysql.status.cache.ssl.misses`**
:   The number of SSL session cache misses.

type: long


**`mysql.status.cache.ssl.size`**
:   The SSL session cache size.

type: long



## table [_table]


## open_cache [_open_cache]

**`mysql.status.cache.table.open_cache.hits`**
:   The number of hits for open tables cache lookups.

type: long


**`mysql.status.cache.table.open_cache.misses`**
:   The number of misses for open tables cache lookups.

type: long


**`mysql.status.cache.table.open_cache.overflows`**
:   Number of times, after a table is opened or closed, a cache instance has an unused entry and the size of the instance is larger than table_open_cache / table_open_cache_instances

type: long



## binlog [_binlog]

**`mysql.status.binlog.cache.disk_use`**
:   type: long


**`mysql.status.binlog.cache.use`**
:   type: long



## bytes [_bytes]

Bytes stats.

**`mysql.status.bytes.received`**
:   The number of bytes received from all clients.

type: long

format: bytes


**`mysql.status.bytes.sent`**
:   The number of bytes sent to all clients.

type: long

format: bytes



## threads [_threads_2]

Threads stats.

**`mysql.status.threads.cached`**
:   The number of cached threads.

type: long


**`mysql.status.threads.created`**
:   The number of created threads.

type: long


**`mysql.status.threads.connected`**
:   The number of connected threads.

type: long


**`mysql.status.threads.running`**
:   The number of running threads.

type: long


**`mysql.status.connections`**
:   type: long



## created [_created]

**`mysql.status.created.tmp.disk_tables`**
:   type: long


**`mysql.status.created.tmp.files`**
:   type: long


**`mysql.status.created.tmp.tables`**
:   type: long



## delayed [_delayed]

**`mysql.status.delayed.errors`**
:   type: long


**`mysql.status.delayed.insert_threads`**
:   type: long


**`mysql.status.delayed.writes`**
:   type: long


**`mysql.status.flush_commands`**
:   type: long


**`mysql.status.max_used_connections`**
:   type: long



## open [_open_2]

**`mysql.status.open.files`**
:   type: long


**`mysql.status.open.streams`**
:   type: long


**`mysql.status.open.tables`**
:   type: long


**`mysql.status.opened_tables`**
:   type: long



## command [_command]

**`mysql.status.command.delete`**
:   The number of DELETE queries since startup.

type: long


**`mysql.status.command.insert`**
:   The number of INSERT queries since startup.

type: long


**`mysql.status.command.select`**
:   The number of SELECT queries since startup.

type: long


**`mysql.status.command.update`**
:   The number of UPDATE queries since startup.

type: long


**`mysql.status.queries`**
:   The number of statements executed by the server. This variable includes statements executed within stored programs, unlike the Questions variable. It does not count COM_PING or COM_STATISTICS commands.

type: long


**`mysql.status.questions`**
:   The number of statements executed by the server. This includes only statements sent to the server by clients and not statements executed within stored programs, unlike the Queries variable. This variable does not count COM_PING, COM_STATISTICS, COM_STMT_PREPARE, COM_STMT_CLOSE, or COM_STMT_RESET commands.

type: long



## handler [_handler]

**`mysql.status.handler.commit`**
:   The number of internal COMMIT statements.

type: long


**`mysql.status.handler.delete`**
:   The number of times that rows have been deleted from tables.

type: long


**`mysql.status.handler.external_lock`**
:   The server increments this variable for each call to its external_lock() function, which generally occurs at the beginning and end of access to a table instance.

type: long


**`mysql.status.handler.mrr_init`**
:   The number of times the server uses a storage engine’s own Multi-Range Read implementation for table access.

type: long


**`mysql.status.handler.prepare`**
:   A counter for the prepare phase of two-phase commit operations.

type: long



## read [_read_6]

**`mysql.status.handler.read.first`**
:   The number of times the first entry in an index was read.

type: long


**`mysql.status.handler.read.key`**
:   The number of requests to read a row based on a key.

type: long


**`mysql.status.handler.read.last`**
:   The number of requests to read the last key in an index.

type: long


**`mysql.status.handler.read.next`**
:   The number of requests to read the next row in key order.

type: long


**`mysql.status.handler.read.prev`**
:   The number of requests to read the previous row in key order.

type: long


**`mysql.status.handler.read.rnd`**
:   The number of requests to read a row based on a fixed position.

type: long


**`mysql.status.handler.read.rnd_next`**
:   The number of requests to read the next row in the data file.

type: long


**`mysql.status.handler.rollback`**
:   The number of requests for a storage engine to perform a rollback operation.

type: long


**`mysql.status.handler.savepoint`**
:   The number of requests for a storage engine to place a savepoint.

type: long


**`mysql.status.handler.savepoint_rollback`**
:   The number of requests for a storage engine to roll back to a savepoint.

type: long


**`mysql.status.handler.update`**
:   The number of requests to update a row in a table.

type: long


**`mysql.status.handler.write`**
:   The number of requests to insert a row in a table.

type: long



## innodb [_innodb]


## rows [_rows]

**`mysql.status.innodb.rows.reads`**
:   The number of rows reads into InnoDB tables.

type: long


**`mysql.status.innodb.rows.inserted`**
:   The number of rows inserted into InnoDB tables.

type: long


**`mysql.status.innodb.rows.deleted`**
:   The number of rows deleted into InnoDB tables.

type: long


**`mysql.status.innodb.rows.updated`**
:   The number of rows updated into InnoDB tables.

type: long



## buffer_pool [_buffer_pool]

**`mysql.status.innodb.buffer_pool.dump_status`**
:   The progress of an operation to record the pages held in the InnoDB buffer pool, triggered by the setting of innodb_buffer_pool_dump_at_shutdown or innodb_buffer_pool_dump_now.

type: long


**`mysql.status.innodb.buffer_pool.load_status`**
:   The progress of an operation to warm up the InnoDB buffer pool by reading in a set of pages corresponding to an earlier point in time, triggered by the setting of innodb_buffer_pool_load_at_startup or innodb_buffer_pool_load_now.

type: long



## bytes [_bytes_2]

**`mysql.status.innodb.buffer_pool.bytes.data`**
:   The total number of bytes in the InnoDB buffer pool containing data.

type: long


**`mysql.status.innodb.buffer_pool.bytes.dirty`**
:   The total current number of bytes held in dirty pages in the InnoDB buffer pool.

type: long



## pages [_pages]

**`mysql.status.innodb.buffer_pool.pages.data`**
:   The number of pages in the InnoDB buffer pool containing data.

type: long


**`mysql.status.innodb.buffer_pool.pages.dirty`**
:   The current number of dirty pages in the InnoDB buffer pool.

type: long


**`mysql.status.innodb.buffer_pool.pages.flushed`**
:   The number of requests to flush pages from the InnoDB buffer pool.

type: long


**`mysql.status.innodb.buffer_pool.pages.free`**
:   The number of free pages in the InnoDB buffer pool.

type: long


**`mysql.status.innodb.buffer_pool.pages.latched`**
:   The number of latched pages in the InnoDB buffer pool.

type: long


**`mysql.status.innodb.buffer_pool.pages.misc`**
:   The number of pages in the InnoDB buffer pool that are busy because they have been allocated for administrative overhead, such as row locks or the adaptive hash index.

type: long


**`mysql.status.innodb.buffer_pool.pages.total`**
:   The total size of the InnoDB buffer pool, in pages.

type: long



## read [_read_7]

**`mysql.status.innodb.buffer_pool.read.ahead`**
:   The number of pages read into the InnoDB buffer pool by the read-ahead background thread.

type: long


**`mysql.status.innodb.buffer_pool.read.ahead_evicted`**
:   The number of pages read into the InnoDB buffer pool by the read-ahead background thread that were subsequently evicted without having been accessed by queries.

type: long


**`mysql.status.innodb.buffer_pool.read.ahead_rnd`**
:   The number of "random" read-aheads initiated by InnoDB.

type: long


**`mysql.status.innodb.buffer_pool.read.requests`**
:   The number of logical read requests.

type: long



## pool [_pool_2]

**`mysql.status.innodb.buffer_pool.pool.reads`**
:   The number of logical reads that InnoDB could not satisfy from the buffer pool, and had to read directly from disk.

type: long


**`mysql.status.innodb.buffer_pool.pool.resize_status`**
:   The status of an operation to resize the InnoDB buffer pool dynamically, triggered by setting the innodb_buffer_pool_size parameter dynamically.

type: long


**`mysql.status.innodb.buffer_pool.pool.wait_free`**
:   Normally, writes to the InnoDB buffer pool happen in the background. When InnoDB needs to read or create a page and no clean pages are available, InnoDB flushes some dirty pages first and waits for that operation to finish. This counter counts instances of these waits.

type: long


**`mysql.status.innodb.buffer_pool.write_requests`**
:   The number of writes done to the InnoDB buffer pool.

type: long


