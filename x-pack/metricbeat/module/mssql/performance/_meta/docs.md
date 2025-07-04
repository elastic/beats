`performance` Metricset fetches information from what’s commonly known as [Performance Counters](https://docs.microsoft.com/en-us/sql/relational-databases/system-dynamic-management-views/sys-dm-os-performance-counters-transact-sql?view=sql-server-2017) in MSSQL.

We fetch the following data:

* **page_splits_per_sec**: Number of page splits per second that occur as the result of overflowing index pages.
* **lock_waits_per_sec**: Number of lock requests per second that required the caller to wait.
* **user_connections**: Total number of user connections
* **transactions**: Total number of transactions
* **active_temp_tables**: Number of temporary tables/table variables in use.
* **connections_reset_per_sec**: Total number of logins started from the connection pool.
* **logins_per_sec**: Total number of logins started per second. This does not include pooled connections.
* **logouts_per_sec**: Total number of logout operations started per second.
* **recompilations_per_sec**: Number of statement recompiles per second. Counts the number of times statement recompiles are triggered. Generally, you want the recompiles to be low.
* **compilations_per_sec**: Number of SQL compilations per second. Indicates the number of times the compile code path is entered. Includes compiles caused by statement-level recompilations in SQL Server. After SQL Server user activity is stable, this value reaches a steady state.
* **batch_requests_per_sec**: Number of Transact-SQL command batches received per second. This statistic is affected by all constraints (such as I/O, number of users, cache size, complexity of requests, and so on). High batch requests mean good throughput.
* **cache_hit.pct**: The ratio is the total number of cache hits divided by the total number of cache lookups over the last few thousand page accesses. After a long period of time, the ratio moves very little. Because reading from the cache is much less expensive than reading from disk, you want this ratio to be high
* **page_life_expectancy.sec**: Indicates the number of seconds a page will stay in the buffer pool without references (in seconds).
* **buffer.checkpoint_pages_per_sec**: Indicates the number of pages flushed to disk per second by a checkpoint or other operation that require all dirty pages to be flushed.
* **buffer.database_pages**: Indicates the number of pages in the buffer pool with database content.
* **buffer.target_pages**: Ideal number of pages in the buffer pool.

