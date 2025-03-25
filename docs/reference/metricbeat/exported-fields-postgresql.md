---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-postgresql.html
---

# PostgreSQL fields [exported-fields-postgresql]

Metrics collected from PostgreSQL servers.


## postgresql [_postgresql]

PostgreSQL metrics.


## activity [_activity]

One document per server process, showing information related to the current activity of that process, such as state and current query. Collected by querying pg_stat_activity.

**`postgresql.activity.database.oid`**
:   OID of the database this backend is connected to.

type: long


**`postgresql.activity.database.name`**
:   Name of the database this backend is connected to.

type: keyword


**`postgresql.activity.pid`**
:   Process ID of this backend.

type: long


**`postgresql.activity.user.id`**
:   OID of the user logged into this backend.

type: long


**`postgresql.activity.user.name`**
:   Name of the user logged into this backend.


**`postgresql.activity.application_name`**
:   Name of the application that is connected to this backend.


**`postgresql.activity.client.address`**
:   IP address of the client connected to this backend.


**`postgresql.activity.client.hostname`**
:   Host name of the connected client, as reported by a reverse DNS lookup of client_addr.


**`postgresql.activity.client.port`**
:   TCP port number that the client is using for communication with this backend, or -1 if a Unix socket is used.

type: long


**`postgresql.activity.backend_type`**
:   Type of the current backend. Possible types are autovacuum launcher, autovacuum worker, logical replication launcher, logical replication worker, parallel worker, background writer, client backend, checkpointer, startup, walreceiver, walsender and walwriter. Extensions may register workers with additional backend types.


**`postgresql.activity.backend_start`**
:   Time when this process was started, i.e., when the client connected to the server.

type: date


**`postgresql.activity.transaction_start`**
:   Time when this process' current transaction was started.

type: date


**`postgresql.activity.query_start`**
:   Time when the currently active query was started, or if state is not active, when the last query was started.

type: date


**`postgresql.activity.state_change`**
:   Time when the state was last changed.

type: date


**`postgresql.activity.waiting`**
:   True if this backend is currently waiting on a lock.

type: boolean


**`postgresql.activity.state`**
:   Current overall state of this backend. Possible values are:

* active: The backend is executing a query.
* idle: The backend is waiting for a new client command.
* idle in transaction: The backend is in a transaction, but is not currently executing a query.
* idle in transaction (aborted): This state is similar to idle in transaction, except one of the statements in the transaction caused an error.
* fastpath function call: The backend is executing a fast-path function.
* disabled: This state is reported if track_activities is disabled in this backend.


**`postgresql.activity.query`**
:   Text of this backend’s most recent query. If state is active this field shows the currently executing query. In all other states, it shows the last query that was executed.


**`postgresql.activity.wait_event`**
:   Wait event name if the backend is currently waiting.


**`postgresql.activity.wait_event_type`**
:   The type of event for which the backend is waiting.



## bgwriter [_bgwriter]

Statistics about the background writer process’s activity. Collected using the pg_stat_bgwriter query.

**`postgresql.bgwriter.checkpoints.scheduled`**
:   Number of scheduled checkpoints that have been performed.

type: long


**`postgresql.bgwriter.checkpoints.requested`**
:   Number of requested checkpoints that have been performed.

type: long


**`postgresql.bgwriter.checkpoints.times.write.ms`**
:   Total amount of time that has been spent in the portion of checkpoint processing where files are written to disk, in milliseconds.

type: float


**`postgresql.bgwriter.checkpoints.times.sync.ms`**
:   Total amount of time that has been spent in the portion of checkpoint processing where files are synchronized to disk, in milliseconds.

type: float


**`postgresql.bgwriter.buffers.checkpoints`**
:   Number of buffers written during checkpoints.

type: long


**`postgresql.bgwriter.buffers.clean`**
:   Number of buffers written by the background writer.

type: long


**`postgresql.bgwriter.buffers.clean_full`**
:   Number of times the background writer stopped a cleaning scan because it had written too many buffers.

type: long


**`postgresql.bgwriter.buffers.backend`**
:   Number of buffers written directly by a backend.

type: long


**`postgresql.bgwriter.buffers.backend_fsync`**
:   Number of times a backend had to execute its own fsync call (normally the background writer handles those even when the backend does its own write)

type: long


**`postgresql.bgwriter.buffers.allocated`**
:   Number of buffers allocated.

type: long


**`postgresql.bgwriter.stats_reset`**
:   Time at which these statistics were last reset.

type: date



## database [_database_2]

One row per database, showing database-wide statistics. Collected by querying pg_stat_database

**`postgresql.database.oid`**
:   OID of the database this backend is connected to, or 0 for shared resources.

type: long


**`postgresql.database.name`**
:   Name of the database this backend is connected to, empty for shared resources.

type: keyword


**`postgresql.database.number_of_backends`**
:   Number of backends currently connected to this database.

type: long


**`postgresql.database.transactions.commit`**
:   Number of transactions in this database that have been committed.

type: long


**`postgresql.database.transactions.rollback`**
:   Number of transactions in this database that have been rolled back.

type: long


**`postgresql.database.blocks.read`**
:   Number of disk blocks read in this database.

type: long


**`postgresql.database.blocks.hit`**
:   Number of times disk blocks were found already in the buffer cache, so that a read was not necessary (this only includes hits in the PostgreSQL buffer cache, not the operating system’s file system cache).

type: long


**`postgresql.database.blocks.time.read.ms`**
:   Time spent reading data file blocks by backends in this database, in milliseconds.

type: double


**`postgresql.database.blocks.time.write.ms`**
:   Time spent writing data file blocks by backends in this database, in milliseconds.

type: double


**`postgresql.database.rows.returned`**
:   Number of rows returned by queries in this database.

type: long


**`postgresql.database.rows.fetched`**
:   Number of rows fetched by queries in this database.

type: long


**`postgresql.database.rows.inserted`**
:   Number of rows inserted by queries in this database.

type: long


**`postgresql.database.rows.updated`**
:   Number of rows updated by queries in this database.

type: long


**`postgresql.database.rows.deleted`**
:   Number of rows deleted by queries in this database.

type: long


**`postgresql.database.conflicts`**
:   Number of queries canceled due to conflicts with recovery in this database.

type: long


**`postgresql.database.temporary.files`**
:   Number of temporary files created by queries in this database. All temporary files are counted, regardless of why the temporary file was created (e.g., sorting or hashing), and regardless of the log_temp_files setting.

type: long


**`postgresql.database.temporary.bytes`**
:   Total amount of data written to temporary files by queries in this database. All temporary files are counted, regardless of why the temporary file was created, and regardless of the log_temp_files setting.

type: long


**`postgresql.database.deadlocks`**
:   Number of deadlocks detected in this database.

type: long


**`postgresql.database.stats_reset`**
:   Time at which these statistics were last reset.

type: date



## statement [_statement]

One document per query per user per database, showing information related invocation of that query, such as cpu usage and total time. Collected by querying pg_stat_statements.

**`postgresql.statement.user.id`**
:   OID of the user logged into the backend that ran the query.

type: long


**`postgresql.statement.database.oid`**
:   OID of the database the query was run on.

type: long


**`postgresql.statement.query.id`**
:   ID of the statement.

type: long


**`postgresql.statement.query.text`**
:   Query text


**`postgresql.statement.query.calls`**
:   Number of times the query has been run.

type: long


**`postgresql.statement.query.rows`**
:   Total number of rows returned by query.

type: long


**`postgresql.statement.query.time.total.ms`**
:   Total number of milliseconds spent running query.

type: float


**`postgresql.statement.query.time.min.ms`**
:   Minimum number of milliseconds spent running query.

type: float


**`postgresql.statement.query.time.max.ms`**
:   Maximum number of milliseconds spent running query.

type: float


**`postgresql.statement.query.time.mean.ms`**
:   Mean number of milliseconds spent running query.

type: long


**`postgresql.statement.query.time.stddev.ms`**
:   Population standard deviation of time spent running query, in milliseconds.

type: long


**`postgresql.statement.query.memory.shared.hit`**
:   Total number of shared block cache hits by the query.

type: long


**`postgresql.statement.query.memory.shared.read`**
:   Total number of shared block cache read by the query.

type: long


**`postgresql.statement.query.memory.shared.dirtied`**
:   Total number of shared block cache dirtied by the query.

type: long


**`postgresql.statement.query.memory.shared.written`**
:   Total number of shared block cache written by the query.

type: long


**`postgresql.statement.query.memory.local.hit`**
:   Total number of local block cache hits by the query.

type: long


**`postgresql.statement.query.memory.local.read`**
:   Total number of local block cache read by the query.

type: long


**`postgresql.statement.query.memory.local.dirtied`**
:   Total number of local block cache dirtied by the query.

type: long


**`postgresql.statement.query.memory.local.written`**
:   Total number of local block cache written by the query.

type: long


**`postgresql.statement.query.memory.temp.read`**
:   Total number of temp block cache read by the query.

type: long


**`postgresql.statement.query.memory.temp.written`**
:   Total number of temp block cache written by the query.

type: long


