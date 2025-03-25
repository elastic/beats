---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-mssql.html
---

# MSSQL fields [exported-fields-mssql]

MS SQL module


## mssql [_mssql]

The root field containing all MSSQL fields


## database [_database]

The database that the metrics is being referred to

**`mssql.database.id`**
:   Unique ID of the database inside MSSQL

type: long


**`mssql.database.name`**
:   Name of the database

type: keyword



## performance [_performance_3]

performance metricset fetches information about the Performance Counters

**`mssql.performance.page_splits_per_sec`**
:   Number of page splits per second that occur as the result of overflowing index pages.

type: long


**`mssql.performance.lock_waits_per_sec`**
:   Number of lock requests per second that required the caller to wait.

type: long


**`mssql.performance.user_connections`**
:   Total number of user connections

type: long


**`mssql.performance.transactions`**
:   Total number of transactions

type: long


**`mssql.performance.active_temp_tables`**
:   Number of temporary tables/table variables in use.

type: long


**`mssql.performance.connections_reset_per_sec`**
:   Total number of logins started from the connection pool.

type: long


**`mssql.performance.logins_per_sec`**
:   Total number of logins started per second. This does not include pooled connections.

type: long


**`mssql.performance.logouts_per_sec`**
:   Total number of logout operations started per second.

type: long


**`mssql.performance.recompilations_per_sec`**
:   Number of statement recompiles per second. Counts the number of times statement recompiles are triggered. Generally, you want the recompiles to be low.

type: long


**`mssql.performance.compilations_per_sec`**
:   Number of SQL compilations per second. Indicates the number of times the compile code path is entered. Includes compiles caused by statement-level recompilations in SQL Server. After SQL Server user activity is stable, this value reaches a steady state.

type: long


**`mssql.performance.batch_requests_per_sec`**
:   Number of Transact-SQL command batches received per second. This statistic is affected by all constraints (such as I/O, number of users, cache size, complexity of requests, and so on). High batch requests mean good throughput.

type: long



## cache_hit [_cache_hit]

Indicates the percentage of pages found in the buffer cache without having to read from disk.

**`mssql.performance.buffer.cache_hit.pct`**
:   The ratio is the total number of cache hits divided by the total number of cache lookups over the last few thousand page accesses. After a long period of time, the ratio moves very little. Because reading from the cache is much less expensive than reading from disk, you want this ratio to be high

type: double



## page_life_expectancy [_page_life_expectancy]

Indicates the number of seconds a page will stay in the buffer pool without references.

**`mssql.performance.buffer.page_life_expectancy.sec`**
:   Indicates the number of seconds a page will stay in the buffer pool without references (in seconds).

type: long


**`mssql.performance.buffer.checkpoint_pages_per_sec`**
:   Indicates the number of pages flushed to disk per second by a checkpoint or other operation that require all dirty pages to be flushed.

type: long


**`mssql.performance.buffer.database_pages`**
:   Indicates the number of pages in the buffer pool with database content.

type: long


**`mssql.performance.buffer.target_pages`**
:   Ideal number of pages in the buffer pool.

type: long



## transaction_log [_transaction_log_2]

transaction_log metricset will fetch information about the operation and transaction log of each database from a MSSQL instance


## space_usage [_space_usage]

Space usage information for the transaction log


## since_last_backup [_since_last_backup]

The amount of space used since the last log backup

**`mssql.transaction_log.space_usage.since_last_backup.bytes`**
:   The amount of space used since the last log backup in bytes

type: long



## total [_total_2]

The size of the log

**`mssql.transaction_log.space_usage.total.bytes`**
:   The size of the log in bytes

type: long



## used [_used]

The occupied size of the log

**`mssql.transaction_log.space_usage.used.bytes`**
:   The occupied size of the log in bytes

type: long


**`mssql.transaction_log.space_usage.used.pct`**
:   A percentage of the occupied size of the log as a percent of the total log size

type: float



## stats [_stats_8]

Returns summary level attributes and information on transaction log files of databases. Use this information for monitoring and diagnostics of transaction log health.


## active_size [_active_size]

Total active transaction log size.

**`mssql.transaction_log.stats.active_size.bytes`**
:   Total active transaction log size in bytes

type: long


**`mssql.transaction_log.stats.backup_time`**
:   Last transaction log backup time.

type: date



## recovery_size [_recovery_size]

Log size since log recovery log sequence number (LSN).

**`mssql.transaction_log.stats.recovery_size.bytes`**
:   Log size in bytes since log recovery log sequence number (LSN).

type: long



## since_last_checkpoint [_since_last_checkpoint]

Log size since last checkpoint log sequence number (LSN).

**`mssql.transaction_log.stats.since_last_checkpoint.bytes`**
:   Log size in bytes since last checkpoint log sequence number (LSN).

type: long



## total_size [_total_size]

Total transaction log size.

**`mssql.transaction_log.stats.total_size.bytes`**
:   Total transaction log size in bytes.

type: long


