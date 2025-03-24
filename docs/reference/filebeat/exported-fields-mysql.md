---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-mysql.html
---

# MySQL fields [exported-fields-mysql]

Module for parsing the MySQL log files.


## mysql [_mysql]

Fields from the MySQL log files.

**`mysql.thread_id`**
:   The connection or thread ID for the query.

type: long



## error [_error_4]

Contains fields from the MySQL error logs.

**`mysql.error.thread_id`**
:   type: alias

alias to: mysql.thread_id


**`mysql.error.level`**
:   type: alias

alias to: log.level


**`mysql.error.message`**
:   type: alias

alias to: message



## slowlog [_slowlog_3]

Contains fields from the MySQL slow logs.

**`mysql.slowlog.lock_time.sec`**
:   The amount of time the query waited for the lock to be available. The value is in seconds, as a floating point number.

type: float


**`mysql.slowlog.rows_sent`**
:   The number of rows returned by the query.

type: long


**`mysql.slowlog.rows_examined`**
:   The number of rows scanned by the query.

type: long


**`mysql.slowlog.rows_affected`**
:   The number of rows modified by the query.

type: long


**`mysql.slowlog.bytes_sent`**
:   The number of bytes sent to client.

type: long

format: bytes


**`mysql.slowlog.bytes_received`**
:   The number of bytes received from client.

type: long

format: bytes


**`mysql.slowlog.query`**
:   The slow query.


**`mysql.slowlog.id`**
:   type: alias

alias to: mysql.thread_id


**`mysql.slowlog.schema`**
:   The schema where the slow query was executed.

type: keyword


**`mysql.slowlog.current_user`**
:   Current authenticated user, used to determine access privileges. Can differ from the value for user.

type: keyword


**`mysql.slowlog.last_errno`**
:   Last SQL error seen.

type: keyword


**`mysql.slowlog.killed`**
:   Code of the reason if the query was killed.

type: keyword


**`mysql.slowlog.query_cache_hit`**
:   Whether the query cache was hit.

type: boolean


**`mysql.slowlog.tmp_table`**
:   Whether a temporary table was used to resolve the query.

type: boolean


**`mysql.slowlog.tmp_table_on_disk`**
:   Whether the query needed temporary tables on disk.

type: boolean


**`mysql.slowlog.tmp_tables`**
:   Number of temporary tables created for this query

type: long


**`mysql.slowlog.tmp_disk_tables`**
:   Number of temporary tables created on disk for this query.

type: long


**`mysql.slowlog.tmp_table_sizes`**
:   Size of temporary tables created for this query.

type: long

format: bytes


**`mysql.slowlog.filesort`**
:   Whether filesort optimization was used.

type: boolean


**`mysql.slowlog.filesort_on_disk`**
:   Whether filesort optimization was used and it needed temporary tables on disk.

type: boolean


**`mysql.slowlog.priority_queue`**
:   Whether a priority queue was used for filesort.

type: boolean


**`mysql.slowlog.full_scan`**
:   Whether a full table scan was needed for the slow query.

type: boolean


**`mysql.slowlog.full_join`**
:   Whether a full join was needed for the slow query (no indexes were used for joins).

type: boolean


**`mysql.slowlog.merge_passes`**
:   Number of merge passes executed for the query.

type: long


**`mysql.slowlog.sort_merge_passes`**
:   Number of merge passes that the sort algorithm has had to do.

type: long


**`mysql.slowlog.sort_range_count`**
:   Number of sorts that were done using ranges.

type: long


**`mysql.slowlog.sort_rows`**
:   Number of sorted rows.

type: long


**`mysql.slowlog.sort_scan_count`**
:   Number of sorts that were done by scanning the table.

type: long


**`mysql.slowlog.log_slow_rate_type`**
:   Type of slow log rate limit, it can be `session` if the rate limit is applied per session, or `query` if it applies per query.

type: keyword


**`mysql.slowlog.log_slow_rate_limit`**
:   Slow log rate limit, a value of 100 means that one in a hundred queries or sessions are being logged.

type: keyword


**`mysql.slowlog.read_first`**
:   The number of times the first entry in an index was read.

type: long


**`mysql.slowlog.read_last`**
:   The number of times the last key in an index was read.

type: long


**`mysql.slowlog.read_key`**
:   The number of requests to read a row based on a key.

type: long


**`mysql.slowlog.read_next`**
:   The number of requests to read the next row in key order.

type: long


**`mysql.slowlog.read_prev`**
:   The number of requests to read the previous row in key order.

type: long


**`mysql.slowlog.read_rnd`**
:   The number of requests to read a row based on a fixed position.

type: long


**`mysql.slowlog.read_rnd_next`**
:   The number of requests to read the next row in the data file.

type: long



## innodb [_innodb]

Contains fields relative to InnoDB engine

**`mysql.slowlog.innodb.trx_id`**
:   Transaction ID

type: keyword


**`mysql.slowlog.innodb.io_r_ops`**
:   Number of page read operations.

type: long


**`mysql.slowlog.innodb.io_r_bytes`**
:   Bytes read during page read operations.

type: long

format: bytes


**`mysql.slowlog.innodb.io_r_wait.sec`**
:   How long it took to read all needed data from storage.

type: long


**`mysql.slowlog.innodb.rec_lock_wait.sec`**
:   How long the query waited for locks.

type: long


**`mysql.slowlog.innodb.queue_wait.sec`**
:   How long the query waited to enter the InnoDB queue and to be executed once in the queue.

type: long


**`mysql.slowlog.innodb.pages_distinct`**
:   Approximated count of pages accessed to execute the query.

type: long


**`mysql.slowlog.user`**
:   type: alias

alias to: user.name


**`mysql.slowlog.host`**
:   type: alias

alias to: source.domain


**`mysql.slowlog.ip`**
:   type: alias

alias to: source.ip


