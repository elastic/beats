---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-oracle.html
---

# Oracle fields [exported-fields-oracle]

Oracle database module


## oracle [_oracle]

Oracle module


## performance [_performance_5]

Performance related metrics on a single database instance

**`oracle.performance.machine`**
:   Operating system machine name

type: keyword


**`oracle.performance.buffer_pool`**
:   Name of the buffer pool in the instance

type: keyword


**`oracle.performance.username`**
:   Oracle username

type: keyword


**`oracle.performance.io_reloads`**
:   Reloads / Pins ratio. A Reload is any PIN of an object that is not the first PIN performed since the object handle was created, and which requires loading the object from disk. Pins are the number of times a PIN was requested for objects of this namespace

type: double


**`oracle.performance.lock_requests`**
:   Average of the ratio between *gethits* and *gets* being *Gethits* the number of times an object’s handle was found in memory and *gets* the number of times a lock was requested for objects of this namespace.

type: long


**`oracle.performance.pin_requests`**
:   Average of all pinhits/pins ratios being *PinHits* the number of times all of the metadata pieces of the library object were found in memory and *pins* the number of times a PIN was requested for objects of this namespace

type: double



## cache [_cache_4]

Statistics about all buffer pools available for the instance

**`oracle.performance.cache.buffer.hit.pct`**
:   The cache hit ratio of the specified buffer pool.

type: double


**`oracle.performance.cache.physical_reads`**
:   Physical reads

type: long



## get [_get]

Buffer pool *get* statistics

**`oracle.performance.cache.get.consistent`**
:   Consistent gets statistic

type: long


**`oracle.performance.cache.get.db_blocks`**
:   Database blocks gotten

type: long



## cursors [_cursors]

Cursors information

**`oracle.performance.cursors.avg`**
:   Average cursors opened by username and machine

type: double


**`oracle.performance.cursors.max`**
:   Max cursors opened by username and machine

type: double


**`oracle.performance.cursors.total`**
:   Total opened cursors by username and machine

type: double



## opened [_opened]

Opened cursors statistic

**`oracle.performance.cursors.opened.current`**
:   Total number of current open cursors

type: long


**`oracle.performance.cursors.opened.total`**
:   Total number of cursors opened since the instance started

type: long



## parse [_parse]

Parses statistic information that occured in the current session

**`oracle.performance.cursors.parse.real`**
:   Real number of parses that occurred: session cursor cache hits - parse count (total)

type: long


**`oracle.performance.cursors.parse.total`**
:   Total number of parse calls (hard and soft). A soft parse is a check on an object already in the shared pool, to verify that the permissions on the underlying object have not changed.

type: long


**`oracle.performance.cursors.session.cache_hits`**
:   Number of hits in the session cursor cache. A hit means that the SQL statement did not have to be reparsed.

type: long


**`oracle.performance.cursors.cache_hit.pct`**
:   Ratio of session cursor cache hits from total number of cursors

type: double



## sysmetric [_sysmetric_2]

Sysmetric related metrics.

**`oracle.sysmetric.session_count`**
:   Session Count.

type: long


**`oracle.sysmetric.average_active_sessions`**
:   Average Active Sessions.

type: double


**`oracle.sysmetric.current_os_load`**
:   Current OS Load.

type: double


**`oracle.sysmetric.physical_reads_per_sec`**
:   Physical Reads Per Second.

type: double


**`oracle.sysmetric.user_transaction_per_sec`**
:   User Transaction Per Second.

type: double


**`oracle.sysmetric.total_table_scans_per_txn`**
:   Total Table Scans Per Transaction.

type: double


**`oracle.sysmetric.physical_writes_per_sec`**
:   Physical Writes Per Second.

type: double


**`oracle.sysmetric.total_index_scans_per_txn`**
:   Total Index Scans Per Transaction.

type: double


**`oracle.sysmetric.host_cpu_utilization_pct`**
:   Host CPU Utilization (%).

type: double


**`oracle.sysmetric.network_traffic_volume_per_sec`**
:   Network Traffic Volume Per Second.

type: double


**`oracle.sysmetric.user_rollbacks_per_sec`**
:   User Rollbacks Per Second.

type: long


**`oracle.sysmetric.cpu_usage_per_sec`**
:   CPU Usage Per Second.

type: double


**`oracle.sysmetric.db_block_changes_per_sec`**
:   DB Block Changes Per Second.

type: double


**`oracle.sysmetric.physical_read_total_bytes_per_sec`**
:   Physical Read Total Bytes Per Second.

type: double


**`oracle.sysmetric.response_time_per_txn`**
:   Response Time Per Transaction.

type: double



## tablespace [_tablespace]

tablespace

**`oracle.tablespace.name`**
:   Tablespace name

type: keyword



## data_file [_data_file]

Database files information

**`oracle.tablespace.data_file.id`**
:   Tablespace unique identifier

type: long


**`oracle.tablespace.data_file.name`**
:   Filename of the data file

type: keyword



## size [_size_3]

Size information about the file

**`oracle.tablespace.data_file.size.max.bytes`**
:   Maximum file size in bytes

type: long

format: bytes


**`oracle.tablespace.data_file.size.bytes`**
:   Size of the file in bytes

type: long

format: bytes


**`oracle.tablespace.data_file.size.free.bytes`**
:   The size of the file available for user data. The actual size of the file minus this value is used to store file related metadata.

type: long

format: bytes


**`oracle.tablespace.data_file.status`**
:   *File status: AVAILABLE or INVALID (INVALID means that the file number is not in use, for example, a file in a tablespace that was dropped)*

type: keyword


**`oracle.tablespace.data_file.online_status`**
:   Last known online status of the data file. One of SYSOFF, SYSTEM, OFFLINE, ONLINE or RECOVER.

type: keyword



## space [_space]

Tablespace space usage information

**`oracle.tablespace.space.free.bytes`**
:   Tablespace total free space available, in bytes.

type: long

format: bytes


**`oracle.tablespace.space.used.bytes`**
:   Tablespace used space, in bytes.

type: long

format: bytes


**`oracle.tablespace.space.total.bytes`**
:   Tablespace total size, in bytes.

type: long

format: bytes


