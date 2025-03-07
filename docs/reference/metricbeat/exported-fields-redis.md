---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-redis.html
---

# Redis fields [exported-fields-redis]

Redis metrics collected from Redis.


## redis [_redis]

`redis` contains the information and statistics from Redis.


## info [_info_6]

`info` contains the information and statistics returned by the `INFO` command.


## clients [_clients]

Redis client stats.

**`redis.info.clients.connected`**
:   Number of client connections (excluding connections from slaves).

type: long


**`redis.info.clients.max_output_buffer`**
:   Longest output list among current client connections.

type: long


**`redis.info.clients.max_input_buffer`**
:   Biggest input buffer among current client connections (on redis 5.0).

type: long


**`redis.info.clients.blocked`**
:   Number of clients pending on a blocking call (BLPOP, BRPOP, BRPOPLPUSH).

type: long



## cluster [_cluster_3]

Redis cluster information.

**`redis.info.cluster.enabled`**
:   Indicates that the Redis cluster is enabled.

type: boolean



## cpu [_cpu_10]

Redis CPU stats

**`redis.info.cpu.used.sys`**
:   System CPU consumed by the Redis server.

type: scaled_float


**`redis.info.cpu.used.sys_children`**
:   System CPU consumed by the background processes.

type: scaled_float


**`redis.info.cpu.used.user`**
:   User CPU consumed by the Redis server.

type: scaled_float


**`redis.info.cpu.used.user_children`**
:   User CPU consumed by the background processes.

type: scaled_float



## memory [_memory_10]

Redis memory stats.

**`redis.info.memory.used.value`**
:   Total number of bytes allocated by Redis.

type: long

format: bytes


**`redis.info.memory.used.rss`**
:   Number of bytes that Redis allocated as seen by the operating system (a.k.a resident set size).

type: long

format: bytes


**`redis.info.memory.used.peak`**
:   Peak memory consumed by Redis.

type: long

format: bytes


**`redis.info.memory.used.lua`**
:   Used memory by the Lua engine.

type: long

format: bytes


**`redis.info.memory.used.dataset`**
:   The size in bytes of the dataset

type: long

format: bytes


**`redis.info.memory.max.value`**
:   Memory limit.

type: long

format: bytes


**`redis.info.memory.max.policy`**
:   Eviction policy to use when memory limit is reached.

type: keyword


**`redis.info.memory.fragmentation.ratio`**
:   Ratio between used_memory_rss and used_memory

type: float


**`redis.info.memory.fragmentation.bytes`**
:   Bytes between used_memory_rss and used_memory

type: long

format: bytes


**`redis.info.memory.active_defrag.is_running`**
:   Flag indicating if active defragmentation is active

type: boolean


**`redis.info.memory.allocator`**
:   Memory allocator.

type: keyword


**`redis.info.memory.allocator_stats.allocated`**
:   Allocated memory

type: long

format: bytes


**`redis.info.memory.allocator_stats.active`**
:   Active memory

type: long

format: bytes


**`redis.info.memory.allocator_stats.resident`**
:   Resident memory

type: long

format: bytes


**`redis.info.memory.allocator_stats.fragmentation.ratio`**
:   Fragmentation ratio

type: float


**`redis.info.memory.allocator_stats.fragmentation.bytes`**
:   Fragmented bytes

type: long

format: bytes


**`redis.info.memory.allocator_stats.rss.ratio`**
:   Resident ratio

type: float


**`redis.info.memory.allocator_stats.rss.bytes`**
:   Resident bytes

type: long

format: bytes



## persistence [_persistence]

Redis CPU stats.

**`redis.info.persistence.loading`**
:   Flag indicating if the load of a dump file is on-going

type: boolean



## rdb [_rdb]

Provides information about RDB persistence

**`redis.info.persistence.rdb.last_save.changes_since`**
:   Number of changes since the last dump

type: long


**`redis.info.persistence.rdb.last_save.time`**
:   Epoch-based timestamp of last successful RDB save

type: long


**`redis.info.persistence.rdb.bgsave.in_progress`**
:   Flag indicating a RDB save is on-going

type: boolean


**`redis.info.persistence.rdb.bgsave.last_status`**
:   Status of the last RDB save operation

type: keyword


**`redis.info.persistence.rdb.bgsave.last_time.sec`**
:   Duration of the last RDB save operation in seconds

type: long

format: duration


**`redis.info.persistence.rdb.bgsave.current_time.sec`**
:   Duration of the on-going RDB save operation if any

type: long

format: duration


**`redis.info.persistence.rdb.copy_on_write.last_size`**
:   The size in bytes of copy-on-write allocations during the last RBD save operation

type: long

format: bytes



## aof [_aof]

Provides information about AOF persitence

**`redis.info.persistence.aof.enabled`**
:   Flag indicating AOF logging is activated

type: boolean


**`redis.info.persistence.aof.rewrite.in_progress`**
:   Flag indicating a AOF rewrite operation is on-going

type: boolean


**`redis.info.persistence.aof.rewrite.scheduled`**
:   Flag indicating an AOF rewrite operation will be scheduled once the on-going RDB save is complete.

type: boolean


**`redis.info.persistence.aof.rewrite.last_time.sec`**
:   Duration of the last AOF rewrite operation in seconds

type: long

format: duration


**`redis.info.persistence.aof.rewrite.current_time.sec`**
:   Duration of the on-going AOF rewrite operation if any

type: long

format: duration


**`redis.info.persistence.aof.rewrite.buffer.size`**
:   Size of the AOF rewrite buffer

type: long

format: bytes


**`redis.info.persistence.aof.bgrewrite.last_status`**
:   Status of the last AOF rewrite operatio

type: keyword


**`redis.info.persistence.aof.write.last_status`**
:   Status of the last write operation to the AOF

type: keyword


**`redis.info.persistence.aof.copy_on_write.last_size`**
:   The size in bytes of copy-on-write allocations during the last RBD save operation

type: long

format: bytes


**`redis.info.persistence.aof.buffer.size`**
:   Size of the AOF buffer

type: long

format: bytes


**`redis.info.persistence.aof.size.current`**
:   AOF current file size

type: long

format: bytes


**`redis.info.persistence.aof.size.base`**
:   AOF file size on latest startup or rewrite

type: long

format: bytes


**`redis.info.persistence.aof.fsync.pending`**
:   Number of fsync pending jobs in background I/O queue

type: long


**`redis.info.persistence.aof.fsync.delayed`**
:   Delayed fsync counter

type: long



## replication [_replication_2]

Replication

**`redis.info.replication.role`**
:   Role of the instance (can be "master", or "slave").

type: keyword


**`redis.info.replication.connected_slaves`**
:   Number of connected slaves

type: long


**`redis.info.replication.backlog.active`**
:   Flag indicating replication backlog is active

type: long


**`redis.info.replication.backlog.size`**
:   Total size in bytes of the replication backlog buffer

type: long

format: bytes


**`redis.info.replication.backlog.first_byte_offset`**
:   The master offset of the replication backlog buffer

type: long


**`redis.info.replication.backlog.histlen`**
:   Size in bytes of the data in the replication backlog buffer

type: long


**`redis.info.replication.master.offset`**
:   The server’s current replication offset

type: long


**`redis.info.replication.master.second_offset`**
:   The offset up to which replication IDs are accepted

type: long


**`redis.info.replication.master.link_status`**
:   Status of the link (up/down)

type: keyword


**`redis.info.replication.master.last_io_seconds_ago`**
:   Number of seconds since the last interaction with master

type: long

format: duration


**`redis.info.replication.master.sync.in_progress`**
:   Indicate the master is syncing to the slave

type: boolean


**`redis.info.replication.master.sync.left_bytes`**
:   Number of bytes left before syncing is complete

type: long

format: bytes


**`redis.info.replication.master.sync.last_io_seconds_ago`**
:   Number of seconds since last transfer I/O during a SYNC operation

type: long

format: duration


**`redis.info.replication.slave.offset`**
:   The replication offset of the slave instance

type: long


**`redis.info.replication.slave.priority`**
:   The priority of the instance as a candidate for failover

type: long


**`redis.info.replication.slave.is_readonly`**
:   Flag indicating if the slave is read-only

type: boolean



## server [_server_9]

Server info

**`redis.info.server.version`**
:   None

type: alias

alias to: service.version


**`redis.info.server.git_sha1`**
:   None

type: keyword


**`redis.info.server.git_dirty`**
:   None

type: keyword


**`redis.info.server.build_id`**
:   None

type: keyword


**`redis.info.server.mode`**
:   None

type: keyword


**`redis.info.server.os`**
:   None

type: alias

alias to: os.full


**`redis.info.server.arch_bits`**
:   None

type: keyword


**`redis.info.server.multiplexing_api`**
:   None

type: keyword


**`redis.info.server.gcc_version`**
:   None

type: keyword


**`redis.info.server.process_id`**
:   None

type: alias

alias to: process.pid


**`redis.info.server.run_id`**
:   None

type: keyword


**`redis.info.server.tcp_port`**
:   None

type: long


**`redis.info.server.uptime`**
:   None

type: long


**`redis.info.server.hz`**
:   None

type: long


**`redis.info.server.lru_clock`**
:   None

type: long


**`redis.info.server.config_file`**
:   None

type: keyword



## stats [_stats_10]

Redis stats.

**`redis.info.stats.connections.received`**
:   Total number of connections received.

type: long


**`redis.info.stats.connections.rejected`**
:   Total number of connections rejected.

type: long


**`redis.info.stats.commands_processed`**
:   Total number of commands processed.

type: long


**`redis.info.stats.net.input.bytes`**
:   Total network input in bytes.

type: long


**`redis.info.stats.net.output.bytes`**
:   Total network output in bytes.

type: long


**`redis.info.stats.instantaneous.ops_per_sec`**
:   Number of commands processed per second

type: long


**`redis.info.stats.instantaneous.input_kbps`**
:   The network’s read rate per second in KB/sec

type: scaled_float


**`redis.info.stats.instantaneous.output_kbps`**
:   The network’s write rate per second in KB/sec

type: scaled_float


**`redis.info.stats.sync.full`**
:   The number of full resyncs with slaves

type: long


**`redis.info.stats.sync.partial.ok`**
:   The number of accepted partial resync requests

type: long


**`redis.info.stats.sync.partial.err`**
:   The number of denied partial resync requests

type: long


**`redis.info.stats.keys.expired`**
:   Total number of key expiration events

type: long


**`redis.info.stats.keys.evicted`**
:   Number of evicted keys due to maxmemory limit

type: long


**`redis.info.stats.keyspace.hits`**
:   Number of successful lookup of keys in the main dictionary

type: long


**`redis.info.stats.keyspace.misses`**
:   Number of failed lookup of keys in the main dictionary

type: long


**`redis.info.stats.pubsub.channels`**
:   Global number of pub/sub channels with client subscriptions

type: long


**`redis.info.stats.pubsub.patterns`**
:   Global number of pub/sub pattern with client subscriptions

type: long


**`redis.info.stats.latest_fork_usec`**
:   Duration of the latest fork operation in microseconds

type: long


**`redis.info.stats.migrate_cached_sockets`**
:   The number of sockets open for MIGRATE purposes

type: long


**`redis.info.stats.slave_expires_tracked_keys`**
:   The number of keys tracked for expiry purposes (applicable only to writable slaves)

type: long


**`redis.info.stats.active_defrag.hits`**
:   Number of value reallocations performed by active the defragmentation process

type: long


**`redis.info.stats.active_defrag.misses`**
:   Number of aborted value reallocations started by the active defragmentation process

type: long


**`redis.info.stats.active_defrag.key_hits`**
:   Number of keys that were actively defragmented

type: long


**`redis.info.stats.active_defrag.key_misses`**
:   Number of keys that were skipped by the active defragmentation process

type: long


**`redis.info.slowlog.count`**
:   Count of slow operations

type: long



## commandstats [_commandstats]

Redis command statistics

**`redis.info.commandstats.*.calls`**
:   The number of calls that reached command execution (not rejected).

type: long


**`redis.info.commandstats.*.usec`**
:   The total CPU time consumed by these commands.

type: long


**`redis.info.commandstats.*.usec_per_call`**
:   The average CPU consumed per command execution.

type: float


**`redis.info.commandstats.*.rejected_calls`**
:   The number of rejected calls (on redis 6.2-rc2).

type: long


**`redis.info.commandstats.*.failed_calls`**
:   The number of failed calls (on redis 6.2-rc2).

type: long



## key [_key_2]

`key` contains information about keys.

**`redis.key.name`**
:   Key name.

type: keyword


**`redis.key.id`**
:   Unique id for this key (With the form <keyspace>:<name>).

type: keyword


**`redis.key.type`**
:   Key type as shown by `TYPE` command.

type: keyword


**`redis.key.length`**
:   Length of the key (Number of elements for lists, length for strings, cardinality for sets).

type: long


**`redis.key.expire.ttl`**
:   Seconds to expire.

type: long



## keyspace [_keyspace]

`keyspace` contains the information about the keyspaces returned by the `INFO` command.

**`redis.keyspace.id`**
:   Keyspace identifier.

type: keyword


**`redis.keyspace.avg_ttl`**
:   Average ttl.

type: long


**`redis.keyspace.keys`**
:   Number of keys in the keyspace.

type: long


**`redis.keyspace.expires`**
:   type: long


