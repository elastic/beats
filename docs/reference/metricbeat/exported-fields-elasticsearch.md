---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-elasticsearch.html
---

# Elasticsearch fields [exported-fields-elasticsearch]

Elasticsearch module

**`cluster_settings.cluster.metadata.display_name`**
:   type: keyword


**`index_recovery.shards.start_time_in_millis`**
:   type: alias

alias to: elasticsearch.index.recovery.start_time.ms


**`index_recovery.shards.stop_time_in_millis`**
:   type: alias

alias to: elasticsearch.index.recovery.stop_time.ms


**`index_recovery.shards.total_time_in_millis`**
:   type: alias

alias to: elasticsearch.index.recovery.total_time.ms


**`stack_stats.apm.found`**
:   type: alias

alias to: elasticsearch.cluster.stats.stack.apm.found


**`stack_stats.xpack.ccr.enabled`**
:   type: alias

alias to: elasticsearch.cluster.stats.stack.xpack.ccr.enabled


**`stack_stats.xpack.ccr.available`**
:   type: alias

alias to: elasticsearch.cluster.stats.stack.xpack.ccr.available


**`license.status`**
:   type: alias

alias to: elasticsearch.cluster.stats.license.status


**`license.type`**
:   type: alias

alias to: elasticsearch.cluster.stats.license.type


**`shard.primary`**
:   type: alias

alias to: elasticsearch.shard.primary


**`shard.state`**
:   type: alias

alias to: elasticsearch.shard.state


**`shard.index`**
:   type: alias

alias to: elasticsearch.index.name


**`shard.node`**
:   type: alias

alias to: elasticsearch.node.id


**`shard.shard`**
:   type: alias

alias to: elasticsearch.shard.number


**`cluster_stats.indices.count`**
:   type: alias

alias to: elasticsearch.cluster.stats.indices.total


**`cluster_stats.indices.shards.total`**
:   type: alias

alias to: elasticsearch.cluster.stats.indices.shards.count


**`cluster_stats.nodes.count.total`**
:   type: alias

alias to: elasticsearch.cluster.stats.nodes.count


**`cluster_stats.nodes.jvm.max_uptime_in_millis`**
:   type: alias

alias to: elasticsearch.cluster.stats.nodes.jvm.max_uptime.ms


**`cluster_stats.nodes.jvm.mem.heap_used_in_bytes`**
:   type: alias

alias to: elasticsearch.cluster.stats.nodes.jvm.memory.heap.used.bytes


**`cluster_stats.nodes.jvm.mem.heap_max_in_bytes`**
:   type: alias

alias to: elasticsearch.cluster.stats.nodes.jvm.memory.heap.max.bytes


**`cluster_state.nodes_hash`**
:   type: alias

alias to: elasticsearch.cluster.stats.state.nodes_hash


**`cluster_state.version`**
:   type: alias

alias to: elasticsearch.cluster.stats.state.version


**`cluster_state.master_node`**
:   type: alias

alias to: elasticsearch.cluster.stats.state.master_node


**`cluster_state.state_uuid`**
:   type: alias

alias to: elasticsearch.cluster.stats.state.state_uuid


**`cluster_state.status`**
:   type: alias

alias to: elasticsearch.cluster.stats.status


**`timestamp`**
:   type: alias

alias to: @timestamp


**`cluster_uuid`**
:   type: alias

alias to: elasticsearch.cluster.id


**`source_node.uuid`**
:   type: alias

alias to: elasticsearch.node.id


**`source_node.name`**
:   type: alias

alias to: elasticsearch.node.name


**`job_stats.job_id`**
:   type: alias

alias to: elasticsearch.ml.job.id


**`job_stats.forecasts_stats.total`**
:   type: alias

alias to: elasticsearch.ml.job.forecasts_stats.total


**`index_stats.index`**
:   type: alias

alias to: elasticsearch.index.name


**`index_stats.primaries.store.size_in_bytes`**
:   type: alias

alias to: elasticsearch.index.primaries.store.size_in_bytes


**`index_stats.primaries.docs.count`**
:   type: alias

alias to: elasticsearch.index.primaries.docs.count


**`index_stats.primaries.segments.count`**
:   type: alias

alias to: elasticsearch.index.primaries.segments.count


**`index_stats.primaries.refresh.total_time_in_millis`**
:   type: alias

alias to: elasticsearch.index.primaries.refresh.total_time_in_millis


**`index_stats.primaries.merges.total_size_in_bytes`**
:   type: alias

alias to: elasticsearch.index.primaries.merges.total_size_in_bytes


**`index_stats.primaries.indexing.index_total`**
:   type: alias

alias to: elasticsearch.index.primaries.indexing.index_total


**`index_stats.primaries.indexing.index_time_in_millis`**
:   type: alias

alias to: elasticsearch.index.primaries.indexing.index_time_in_millis


**`index_stats.primaries.indexing.throttle_time_in_millis`**
:   type: alias

alias to: elasticsearch.index.primaries.indexing.throttle_time_in_millis


**`index_stats.total.query_cache.memory_size_in_bytes`**
:   type: alias

alias to: elasticsearch.index.total.query_cache.memory_size_in_bytes


**`index_stats.total.fielddata.memory_size_in_bytes`**
:   type: alias

alias to: elasticsearch.index.total.fielddata.memory_size_in_bytes


**`index_stats.total.request_cache.memory_size_in_bytes`**
:   type: alias

alias to: elasticsearch.index.total.request_cache.memory_size_in_bytes


**`index_stats.total.merges.total_size_in_bytes`**
:   type: alias

alias to: elasticsearch.index.total.merges.total_size_in_bytes


**`index_stats.total.refresh.total_time_in_millis`**
:   type: alias

alias to: elasticsearch.index.total.refresh.total_time_in_millis


**`index_stats.total.store.size_in_bytes`**
:   type: alias

alias to: elasticsearch.index.total.store.size_in_bytes


**`index_stats.total.indexing.index_total`**
:   type: alias

alias to: elasticsearch.index.total.indexing.index_total


**`index_stats.total.indexing.index_time_in_millis`**
:   type: alias

alias to: elasticsearch.index.total.indexing.index_time_in_millis


**`index_stats.total.indexing.throttle_time_in_millis`**
:   type: alias

alias to: elasticsearch.index.total.indexing.throttle_time_in_millis


**`index_stats.total.search.query_total`**
:   type: alias

alias to: elasticsearch.index.total.search.query_total


**`index_stats.total.search.query_time_in_millis`**
:   type: alias

alias to: elasticsearch.index.total.search.query_time_in_millis


**`index_stats.total.segments.terms_memory_in_bytes`**
:   type: alias

alias to: elasticsearch.index.total.segments.terms_memory_in_bytes


**`index_stats.total.segments.points_memory_in_bytes`**
:   type: alias

alias to: elasticsearch.index.total.segments.points_memory_in_bytes


**`index_stats.total.segments.count`**
:   type: alias

alias to: elasticsearch.index.total.segments.count


**`index_stats.total.segments.doc_values_memory_in_bytes`**
:   type: alias

alias to: elasticsearch.index.total.segments.doc_values_memory_in_bytes


**`index_stats.total.segments.norms_memory_in_bytes`**
:   type: alias

alias to: elasticsearch.index.total.segments.norms_memory_in_bytes


**`index_stats.total.segments.stored_fields_memory_in_bytes`**
:   type: alias

alias to: elasticsearch.index.total.segments.stored_fields_memory_in_bytes


**`index_stats.total.segments.fixed_bit_set_memory_in_bytes`**
:   type: alias

alias to: elasticsearch.index.total.segments.fixed_bit_set_memory_in_bytes


**`index_stats.total.segments.term_vectors_memory_in_bytes`**
:   type: alias

alias to: elasticsearch.index.total.segments.term_vectors_memory_in_bytes


**`index_stats.total.segments.version_map_memory_in_bytes`**
:   type: alias

alias to: elasticsearch.index.total.segments.version_map_memory_in_bytes


**`index_stats.total.segments.index_writer_memory_in_bytes`**
:   type: alias

alias to: elasticsearch.index.total.segments.index_writer_memory_in_bytes


**`index_stats.total.segments.memory_in_bytes`**
:   type: alias

alias to: elasticsearch.index.total.segments.memory_in_bytes


**`ccr_auto_follow_stats.number_of_failed_follow_indices`**
:   type: alias

alias to: elasticsearch.ccr.auto_follow.failed.follow_indices.count


**`ccr_auto_follow_stats.number_of_failed_remote_cluster_state_requests`**
:   type: alias

alias to: elasticsearch.ccr.auto_follow.failed.remote_cluster_state_requests.count


**`ccr_auto_follow_stats.number_of_successful_follow_indices`**
:   type: alias

alias to: elasticsearch.ccr.auto_follow.success.follow_indices.count


**`ccr_auto_follow_stats.follower.failed_read_requests`**
:   type: alias

alias to: elasticsearch.ccr.requests.failed.read.count


**`ccr_stats.shard_id`**
:   type: alias

alias to: elasticsearch.ccr.follower.shard.number


**`ccr_stats.remote_cluster`**
:   type: alias

alias to: elasticsearch.ccr.remote_cluster


**`ccr_stats.leader_index`**
:   type: alias

alias to: elasticsearch.ccr.leader.index


**`ccr_stats.follower_index`**
:   type: alias

alias to: elasticsearch.ccr.follower.index


**`ccr_stats.leader_global_checkpoint`**
:   type: alias

alias to: elasticsearch.ccr.leader.global_checkpoint


**`ccr_stats.leader_max_seq_no`**
:   type: alias

alias to: elasticsearch.ccr.leader.max_seq_no


**`ccr_stats.follower_global_checkpoint`**
:   type: alias

alias to: elasticsearch.ccr.follower.global_checkpoint


**`ccr_stats.follower_max_seq_no`**
:   type: alias

alias to: elasticsearch.ccr.follower.max_seq_no


**`ccr_stats.last_requested_seq_no`**
:   type: alias

alias to: elasticsearch.ccr.last_requested_seq_no


**`ccr_stats.outstanding_read_requests`**
:   type: alias

alias to: elasticsearch.ccr.requests.outstanding.read.count


**`ccr_stats.outstanding_write_requests`**
:   type: alias

alias to: elasticsearch.ccr.requests.outstanding.write.count


**`ccr_stats.write_buffer_operation_count`**
:   type: alias

alias to: elasticsearch.ccr.write_buffer.operation.count


**`ccr_stats.write_buffer_size_in_bytes`**
:   type: alias

alias to: elasticsearch.ccr.write_buffer.size.bytes


**`ccr_stats.follower_mapping_version`**
:   type: alias

alias to: elasticsearch.ccr.follower.mapping_version


**`ccr_stats.follower_settings_version`**
:   type: alias

alias to: elasticsearch.ccr.follower.settings_version


**`ccr_stats.follower_aliases_version`**
:   type: alias

alias to: elasticsearch.ccr.follower.aliases_version


**`ccr_stats.total_read_time_millis`**
:   type: alias

alias to: elasticsearch.ccr.total_time.read.ms


**`ccr_stats.total_read_remote_exec_time_millis`**
:   type: alias

alias to: elasticsearch.ccr.total_time.read.remote_exec.ms


**`ccr_stats.successful_read_requests`**
:   type: alias

alias to: elasticsearch.ccr.requests.successful.read.count


**`ccr_stats.failed_read_requests`**
:   type: alias

alias to: elasticsearch.ccr.requests.failed.read.count


**`ccr_stats.operations_read`**
:   type: alias

alias to: elasticsearch.ccr.follower.operations.read.count


**`ccr_stats.operations_written`**
:   type: alias

alias to: elasticsearch.ccr.follower.operations_written


**`ccr_stats.bytes_read`**
:   type: alias

alias to: elasticsearch.ccr.bytes_read


**`ccr_stats.total_write_time_millis`**
:   type: alias

alias to: elasticsearch.ccr.total_time.write.ms


**`ccr_stats.successful_write_requests`**
:   type: alias

alias to: elasticsearch.ccr.requests.successful.write.count


**`ccr_stats.failed_write_requests`**
:   type: alias

alias to: elasticsearch.ccr.requests.failed.write.count


**`node_stats.fs.total.available_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.fs.summary.available.bytes


**`node_stats.fs.total.total_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.fs.summary.total.bytes


**`node_stats.fs.summary.available.bytes`**
:   type: alias

alias to: elasticsearch.node.stats.fs.summary.available.bytes


**`node_stats.fs.summary.total.bytes`**
:   type: alias

alias to: elasticsearch.node.stats.fs.summary.total.bytes


**`node_stats.fs.io_stats.total.operations`**
:   type: alias

alias to: elasticsearch.node.stats.fs.io_stats.total.operations.count


**`node_stats.fs.io_stats.total.read_operations`**
:   type: alias

alias to: elasticsearch.node.stats.fs.io_stats.total.read.operations.count


**`node_stats.fs.io_stats.total.write_operations`**
:   type: alias

alias to: elasticsearch.node.stats.fs.io_stats.total.write.operations.count


**`node_stats.indices.store.size_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.indices.store.size.bytes


**`node_stats.indices.store.size.bytes`**
:   type: alias

alias to: elasticsearch.node.stats.indices.store.size.bytes


**`node_stats.indices.docs.count`**
:   type: alias

alias to: elasticsearch.node.stats.indices.docs.count


**`node_stats.indices.indexing.index_time_in_millis`**
:   type: alias

alias to: elasticsearch.node.stats.indices.indexing.index_time.ms


**`node_stats.indices.indexing.index_total`**
:   type: alias

alias to: elasticsearch.node.stats.indices.indexing.index_total.count


**`node_stats.indices.indexing.throttle_time_in_millis`**
:   type: alias

alias to: elasticsearch.node.stats.indices.indexing.throttle_time.ms


**`node_stats.indices.fielddata.memory_size_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.indices.fielddata.memory.bytes


**`node_stats.indices.query_cache.memory_size_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.indices.query_cache.memory.bytes


**`node_stats.indices.request_cache.memory_size_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.indices.request_cache.memory.bytes


**`node_stats.indices.search.query_time_in_millis`**
:   type: alias

alias to: elasticsearch.node.stats.indices.search.query_time.ms


**`node_stats.indices.search.query_total`**
:   type: alias

alias to: elasticsearch.node.stats.indices.search.query_total.count


**`node_stats.indices.segments.count`**
:   type: alias

alias to: elasticsearch.node.stats.indices.segments.count


**`node_stats.indices.segments.doc_values_memory_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.indices.segments.doc_values.memory.bytes


**`node_stats.indices.segments.fixed_bit_set_memory_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.indices.segments.fixed_bit_set.memory.bytes


**`node_stats.indices.segments.index_writer_memory_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.indices.segments.index_writer.memory.bytes


**`node_stats.indices.segments.memory_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.indices.segments.memory.bytes


**`node_stats.indices.segments.norms_memory_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.indices.segments.norms.memory.bytes


**`node_stats.indices.segments.points_memory_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.indices.segments.points.memory.bytes


**`node_stats.indices.segments.stored_fields_memory_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.indices.segments.stored_fields.memory.bytes


**`node_stats.indices.segments.term_vectors_memory_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.indices.segments.term_vectors.memory.bytes


**`node_stats.indices.segments.terms_memory_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.indices.segments.terms.memory.bytes


**`node_stats.indices.segments.version_map_memory_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.indices.segments.version_map.memory.bytes


**`node_stats.jvm.gc.collectors.old.collection_count`**
:   type: alias

alias to: elasticsearch.node.stats.jvm.gc.collectors.old.collection.count


**`node_stats.jvm.gc.collectors.old.collection_time_in_millis`**
:   type: alias

alias to: elasticsearch.node.stats.jvm.gc.collectors.old.collection.ms


**`node_stats.jvm.gc.collectors.young.collection_count`**
:   type: alias

alias to: elasticsearch.node.stats.jvm.gc.collectors.young.collection.count


**`node_stats.jvm.gc.collectors.young.collection_time_in_millis`**
:   type: alias

alias to: elasticsearch.node.stats.jvm.gc.collectors.young.collection.ms


**`node_stats.jvm.mem.heap_max_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.jvm.mem.heap.max.bytes


**`node_stats.jvm.mem.heap_used_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.jvm.mem.heap.used.bytes


**`node_stats.jvm.mem.heap_used_percent`**
:   type: alias

alias to: elasticsearch.node.stats.jvm.mem.heap.used.pct


**`node_stats.node_id`**
:   type: alias

alias to: elasticsearch.node.id


**`node_stats.os.cpu.load_average.1m`**
:   type: alias

alias to: elasticsearch.node.stats.os.cpu.load_avg.1m


**`node_stats.os.cgroup.cpuacct.usage_nanos`**
:   type: alias

alias to: elasticsearch.node.stats.os.cgroup.cpuacct.usage.ns


**`node_stats.os.cgroup.cpu.cfs_quota_micros`**
:   type: alias

alias to: elasticsearch.node.stats.os.cgroup.cpu.cfs.quota.us


**`node_stats.os.cgroup.cpu.stat.number_of_elapsed_periods`**
:   type: alias

alias to: elasticsearch.node.stats.os.cgroup.cpu.stat.elapsed_periods.count


**`node_stats.os.cgroup.cpu.stat.number_of_times_throttled`**
:   type: alias

alias to: elasticsearch.node.stats.os.cgroup.cpu.stat.times_throttled.count


**`node_stats.os.cgroup.cpu.stat.time_throttled_nanos`**
:   type: alias

alias to: elasticsearch.node.stats.os.cgroup.cpu.stat.time_throttled.ns


**`node_stats.os.cgroup.memory.control_group`**
:   type: alias

alias to: elasticsearch.node.stats.os.cgroup.memory.control_group


**`node_stats.os.cgroup.memory.limit_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.os.cgroup.memory.limit.bytes


**`node_stats.os.cgroup.memory.usage_in_bytes`**
:   type: alias

alias to: elasticsearch.node.stats.os.cgroup.memory.usage.bytes


**`node_stats.process.cpu.percent`**
:   type: alias

alias to: elasticsearch.node.stats.process.cpu.pct


**`node_stats.thread_pool.bulk.queue`**
:   type: alias

alias to: elasticsearch.node.stats.thread_pool.bulk.queue.count


**`node_stats.thread_pool.bulk.rejected`**
:   type: alias

alias to: elasticsearch.node.stats.thread_pool.bulk.rejected.count


**`node_stats.thread_pool.get.queue`**
:   type: alias

alias to: elasticsearch.node.stats.thread_pool.get.queue.count


**`node_stats.thread_pool.get.rejected`**
:   type: alias

alias to: elasticsearch.node.stats.thread_pool.get.rejected.count


**`node_stats.thread_pool.index.queue`**
:   type: alias

alias to: elasticsearch.node.stats.thread_pool.index.queue.count


**`node_stats.thread_pool.index.rejected`**
:   type: alias

alias to: elasticsearch.node.stats.thread_pool.index.rejected.count


**`node_stats.thread_pool.search.queue`**
:   type: alias

alias to: elasticsearch.node.stats.thread_pool.search.queue.count


**`node_stats.thread_pool.search.rejected`**
:   type: alias

alias to: elasticsearch.node.stats.thread_pool.search.rejected.count


**`node_stats.thread_pool.write.queue`**
:   type: alias

alias to: elasticsearch.node.stats.thread_pool.write.queue.count


**`node_stats.thread_pool.write.rejected`**
:   type: alias

alias to: elasticsearch.node.stats.thread_pool.write.rejected.count


**`indices_stats._all.primaries.indexing.index_total`**
:   type: alias

alias to: elasticsearch.index.summary.primaries.indexing.index.count


**`indices_stats._all.primaries.indexing.index_time_in_millis`**
:   type: alias

alias to: elasticsearch.index.summary.primaries.indexing.index.time.ms


**`indices_stats._all.total.search.query_total`**
:   type: alias

alias to: elasticsearch.index.summary.total.search.query.count


**`indices_stats._all.total.search.query_time_in_millis`**
:   type: alias

alias to: elasticsearch.index.summary.total.search.query.time.ms


**`indices_stats._all.total.indexing.index_total`**
:   type: alias

alias to: elasticsearch.index.summary.total.indexing.index.count


**`elasticsearch.cluster.name`**
:   Elasticsearch cluster name.

type: keyword


**`elasticsearch.cluster.id`**
:   Elasticsearch cluster id.

type: keyword


**`elasticsearch.cluster.state.id`**
:   Elasticsearch state id.

type: keyword


**`elasticsearch.node.id`**
:   Node ID

type: keyword


**`elasticsearch.node.name`**
:   Node name.

type: keyword


**`elasticsearch.node.roles`**
:   Node roles.

type: keyword


**`elasticsearch.node.master`**
:   Is the node the master node?

type: boolean


**`elasticsearch.node.mlockall`**
:   Is mlockall enabled on the node?

type: boolean



## ccr [_ccr]

Cross-cluster replication stats

**`elasticsearch.ccr.remote_cluster`**
:   type: keyword


**`elasticsearch.ccr.bytes_read`**
:   type: long


**`elasticsearch.ccr.last_requested_seq_no`**
:   type: long


**`elasticsearch.ccr.shard_id`**
:   type: integer


**`elasticsearch.ccr.total_time.read.ms`**
:   type: long


**`elasticsearch.ccr.total_time.read.remote_exec.ms`**
:   type: long


**`elasticsearch.ccr.total_time.write.ms`**
:   type: long


**`elasticsearch.ccr.read_exceptions`**
:   type: nested


**`elasticsearch.ccr.requests.successful.read.count`**
:   type: long


**`elasticsearch.ccr.requests.successful.write.count`**
:   type: long


**`elasticsearch.ccr.requests.failed.read.count`**
:   type: long


**`elasticsearch.ccr.requests.failed.write.count`**
:   type: long


**`elasticsearch.ccr.requests.outstanding.read.count`**
:   type: long


**`elasticsearch.ccr.requests.outstanding.write.count`**
:   type: long


**`elasticsearch.ccr.write_buffer.size.bytes`**
:   type: long


**`elasticsearch.ccr.write_buffer.operation.count`**
:   type: long


**`elasticsearch.ccr.auto_follow.failed.follow_indices.count`**
:   type: long


**`elasticsearch.ccr.auto_follow.failed.remote_cluster_state_requests.count`**
:   type: long


**`elasticsearch.ccr.auto_follow.success.follow_indices.count`**
:   type: long


**`elasticsearch.ccr.leader.index`**
:   Name of leader index

type: keyword


**`elasticsearch.ccr.leader.max_seq_no`**
:   Maximum sequence number of operation on the leader shard

type: long


**`elasticsearch.ccr.leader.global_checkpoint`**
:   type: long


**`elasticsearch.ccr.follower.index`**
:   Name of follower index

type: keyword


**`elasticsearch.ccr.follower.shard.number`**
:   Number of the shard within the index

type: long


**`elasticsearch.ccr.follower.operations_written`**
:   Number of operations indexed (replicated) into the follower shard from the leader shard

type: long


**`elasticsearch.ccr.follower.time_since_last_read.ms`**
:   Time, in ms, since the follower last fetched from the leader

type: long


**`elasticsearch.ccr.follower.global_checkpoint`**
:   Global checkpoint value on follower shard

type: long


**`elasticsearch.ccr.follower.max_seq_no`**
:   Maximum sequence number of operation on the follower shard

type: long


**`elasticsearch.ccr.follower.mapping_version`**
:   type: long


**`elasticsearch.ccr.follower.settings_version`**
:   type: long


**`elasticsearch.ccr.follower.aliases_version`**
:   type: long


**`elasticsearch.ccr.follower.operations.read.count`**
:   type: long



## cluster.stats [_cluster_stats]

Cluster stats

**`elasticsearch.cluster.stats.version`**
:   type: keyword


**`elasticsearch.cluster.stats.state.nodes_hash`**
:   type: keyword


**`elasticsearch.cluster.stats.state.master_node`**
:   type: keyword


**`elasticsearch.cluster.stats.state.version`**
:   type: keyword


**`elasticsearch.cluster.stats.state.state_uuid`**
:   type: keyword


**`elasticsearch.cluster.stats.state.nodes`**
:   type: flattened


**`elasticsearch.cluster.stats.status`**
:   Cluster status (green, yellow, red).

type: keyword



## nodes [_nodes]

Nodes statistics.

**`elasticsearch.cluster.stats.nodes.fs.total.bytes`**
:   type: long


**`elasticsearch.cluster.stats.nodes.fs.available.bytes`**
:   type: long


**`elasticsearch.cluster.stats.nodes.count`**
:   Total number of nodes in cluster.

type: long


**`elasticsearch.cluster.stats.nodes.master`**
:   Number of master-eligible nodes in cluster.

type: long


**`elasticsearch.cluster.stats.nodes.data`**
:   Number of data nodes in cluster.

type: long


**`elasticsearch.cluster.stats.nodes.jvm.max_uptime.ms`**
:   type: long


**`elasticsearch.cluster.stats.nodes.jvm.memory.heap.max.bytes`**
:   type: long


**`elasticsearch.cluster.stats.nodes.jvm.memory.heap.used.bytes`**
:   type: long



## indices [_indices]

Indices statistics.

**`elasticsearch.cluster.stats.indices.store.size.bytes`**
:   type: long


**`elasticsearch.cluster.stats.indices.store.total_data_set_size.bytes`**
:   type: long


**`elasticsearch.cluster.stats.indices.total`**
:   Total number of indices in cluster.

type: long



## shards [_shards]

Shard statistics.

**`elasticsearch.cluster.stats.indices.shards.docs.total`**
:   type: long


**`elasticsearch.cluster.stats.indices.shards.count`**
:   Total number of shards in cluster.

type: long


**`elasticsearch.cluster.stats.indices.shards.primaries`**
:   Total number of primary shards in cluster.

type: long


**`elasticsearch.cluster.stats.indices.fielddata.memory.bytes`**
:   Memory used for fielddata.

type: long


**`elasticsearch.cluster.stats.license.expiry_date_in_millis`**
:   type: long


**`elasticsearch.cluster.stats.license.status`**
:   type: keyword


**`elasticsearch.cluster.stats.license.type`**
:   type: keyword


**`elasticsearch.cluster.stats.stack.apm.found`**
:   type: boolean


**`elasticsearch.cluster.stats.stack.xpack.ccr.available`**
:   type: boolean


**`elasticsearch.cluster.stats.stack.xpack.ccr.enabled`**
:   type: boolean



## enrich [_enrich]

Enrich stats

**`elasticsearch.enrich.executing_policy.name`**
:   type: keyword


**`elasticsearch.enrich.executing_policy.task.id`**
:   type: long


**`elasticsearch.enrich.executing_policy.task.task`**
:   type: keyword


**`elasticsearch.enrich.executing_policy.task.action`**
:   type: keyword


**`elasticsearch.enrich.executing_policy.task.cancellable`**
:   type: boolean


**`elasticsearch.enrich.executing_policy.task.parent_task_id`**
:   type: keyword


**`elasticsearch.enrich.executing_policy.task.time.start.ms`**
:   type: long


**`elasticsearch.enrich.executing_policy.task.time.running.nano`**
:   type: long


**`elasticsearch.enrich.queue.size`**
:   Number of search requests in the queue.

type: long


**`elasticsearch.enrich.executed_searches.total`**
:   Number of search requests that enrich processors have executed since node startup.

type: long


**`elasticsearch.enrich.remote_requests.current`**
:   Current number of outstanding remote requests.

type: long


**`elasticsearch.enrich.remote_requests.total`**
:   Number of outstanding remote requests executed since node startup.

type: long



## index [_index_3]

index

**`elasticsearch.index.hidden`**
:   type: boolean


**`elasticsearch.index.shards.total`**
:   type: long


**`elasticsearch.index.shards.primaries`**
:   type: long


**`elasticsearch.index.uuid`**
:   type: keyword


**`elasticsearch.index.status`**
:   type: keyword


**`elasticsearch.index.tier_preference`**
:   type: keyword


**`elasticsearch.index.creation_date`**
:   type: date


**`elasticsearch.index.version`**
:   type: keyword


**`elasticsearch.index.name`**
:   Index name.

type: keyword


**`elasticsearch.index.primaries.search.query_total`**
:   type: long


**`elasticsearch.index.primaries.search.query_time_in_millis`**
:   type: long


**`elasticsearch.index.primaries.request_cache.memory_size_in_bytes`**
:   type: long


**`elasticsearch.index.primaries.request_cache.evictions`**
:   type: long


**`elasticsearch.index.primaries.request_cache.hit_count`**
:   type: long


**`elasticsearch.index.primaries.request_cache.miss_count`**
:   type: long


**`elasticsearch.index.primaries.query_cache.memory_size_in_bytes`**
:   type: long


**`elasticsearch.index.primaries.query_cache.hit_count`**
:   type: long


**`elasticsearch.index.primaries.query_cache.miss_count`**
:   type: long


**`elasticsearch.index.primaries.store.size_in_bytes`**
:   type: long


**`elasticsearch.index.primaries.store.total_data_set_size_in_bytes`**
:   type: long


**`elasticsearch.index.primaries.docs.count`**
:   type: long


**`elasticsearch.index.primaries.docs.deleted`**
:   type: long


**`elasticsearch.index.primaries.segments.count`**
:   type: long


**`elasticsearch.index.primaries.segments.memory_in_bytes`**
:   type: long


**`elasticsearch.index.primaries.segments.terms_memory_in_bytes`**
:   type: long


**`elasticsearch.index.primaries.segments.stored_fields_memory_in_bytes`**
:   type: long


**`elasticsearch.index.primaries.segments.term_vectors_memory_in_bytes`**
:   type: long


**`elasticsearch.index.primaries.segments.norms_memory_in_bytes`**
:   type: long


**`elasticsearch.index.primaries.segments.points_memory_in_bytes`**
:   type: long


**`elasticsearch.index.primaries.segments.doc_values_memory_in_bytes`**
:   type: long


**`elasticsearch.index.primaries.segments.index_writer_memory_in_bytes`**
:   type: long


**`elasticsearch.index.primaries.segments.version_map_memory_in_bytes`**
:   type: long


**`elasticsearch.index.primaries.segments.fixed_bit_set_memory_in_bytes`**
:   type: long


**`elasticsearch.index.primaries.refresh.total_time_in_millis`**
:   type: long


**`elasticsearch.index.primaries.refresh.external_total_time_in_millis`**
:   type: long


**`elasticsearch.index.primaries.merges.total_size_in_bytes`**
:   type: long


**`elasticsearch.index.primaries.indexing.index_total`**
:   type: long


**`elasticsearch.index.primaries.indexing.index_time_in_millis`**
:   type: long


**`elasticsearch.index.primaries.indexing.throttle_time_in_millis`**
:   type: long


**`elasticsearch.index.total.docs.count`**
:   Total number of documents in the index.

type: long


**`elasticsearch.index.total.docs.deleted`**
:   Total number of deleted documents in the index.

type: long


**`elasticsearch.index.total.store.size_in_bytes`**
:   Total size of the index in bytes.

type: long

format: bytes


**`elasticsearch.index.total.store.total_data_set_size_in_bytes`**
:   Total size of the index in bytes including backing data for partially mounted indices.

type: long

format: bytes


**`elasticsearch.index.total.query_cache.memory_size_in_bytes`**
:   type: long


**`elasticsearch.index.total.query_cache.evictions`**
:   type: long


**`elasticsearch.index.total.query_cache.hit_count`**
:   type: long


**`elasticsearch.index.total.query_cache.miss_count`**
:   type: long


**`elasticsearch.index.total.fielddata.memory_size_in_bytes`**
:   type: long


**`elasticsearch.index.total.fielddata.evictions`**
:   type: long


**`elasticsearch.index.total.request_cache.memory_size_in_bytes`**
:   type: long


**`elasticsearch.index.total.request_cache.evictions`**
:   type: long


**`elasticsearch.index.total.request_cache.hit_count`**
:   type: long


**`elasticsearch.index.total.request_cache.miss_count`**
:   type: long


**`elasticsearch.index.total.merges.total_size_in_bytes`**
:   type: long


**`elasticsearch.index.total.refresh.total_time_in_millis`**
:   type: long


**`elasticsearch.index.total.refresh.external_total_time_in_millis`**
:   type: long


**`elasticsearch.index.total.segments.memory_in_bytes`**
:   Total number of memory used by the segments in bytes.

type: long

format: bytes


**`elasticsearch.index.total.segments.terms_memory_in_bytes`**
:   type: long


**`elasticsearch.index.total.segments.points_memory_in_bytes`**
:   type: long


**`elasticsearch.index.total.segments.count`**
:   Total number of index segments.

type: long


**`elasticsearch.index.total.segments.doc_values_memory_in_bytes`**
:   type: long


**`elasticsearch.index.total.segments.norms_memory_in_bytes`**
:   type: long


**`elasticsearch.index.total.segments.stored_fields_memory_in_bytes`**
:   type: long


**`elasticsearch.index.total.segments.fixed_bit_set_memory_in_bytes`**
:   type: long


**`elasticsearch.index.total.segments.term_vectors_memory_in_bytes`**
:   type: long


**`elasticsearch.index.total.segments.version_map_memory_in_bytes`**
:   type: long


**`elasticsearch.index.total.segments.index_writer_memory_in_bytes`**
:   type: long


**`elasticsearch.index.total.search.query_total`**
:   type: long


**`elasticsearch.index.total.search.query_time_in_millis`**
:   type: long


**`elasticsearch.index.total.indexing.index_total`**
:   type: long


**`elasticsearch.index.total.indexing.index_time_in_millis`**
:   type: long


**`elasticsearch.index.total.indexing.throttle_time_in_millis`**
:   type: long


**`elasticsearch.index.total.bulk.total_size_in_bytes`**
:   type: long


**`elasticsearch.index.total.bulk.avg_size_in_bytes`**
:   type: long


**`elasticsearch.index.total.bulk.avg_time_in_millis`**
:   type: long


**`elasticsearch.index.total.bulk.total_operations`**
:   type: long


**`elasticsearch.index.total.bulk.total_time_in_millis`**
:   type: long



## index.recovery [_index_recovery]

index

**`elasticsearch.index.recovery.index.files.percent`**
:   type: keyword


**`elasticsearch.index.recovery.index.files.recovered`**
:   type: long


**`elasticsearch.index.recovery.index.files.reused`**
:   type: long


**`elasticsearch.index.recovery.index.files.total`**
:   type: long


**`elasticsearch.index.recovery.index.size.recovered_in_bytes`**
:   type: long


**`elasticsearch.index.recovery.index.size.reused_in_bytes`**
:   type: long


**`elasticsearch.index.recovery.index.size.total_in_bytes`**
:   type: long


**`elasticsearch.index.recovery.name`**
:   type: keyword


**`elasticsearch.index.recovery.total_time.ms`**
:   type: long


**`elasticsearch.index.recovery.stop_time.ms`**
:   type: long


**`elasticsearch.index.recovery.start_time.ms`**
:   type: long


**`elasticsearch.index.recovery.id`**
:   Shard recovery id.

type: long


**`elasticsearch.index.recovery.type`**
:   Shard recovery type.

type: keyword


**`elasticsearch.index.recovery.primary`**
:   True if primary shard.

type: boolean


**`elasticsearch.index.recovery.stage`**
:   Recovery stage.

type: keyword


**`elasticsearch.index.recovery.translog.percent`**
:   type: keyword


**`elasticsearch.index.recovery.translog.total`**
:   type: long


**`elasticsearch.index.recovery.translog.total_on_start`**
:   type: long


**`elasticsearch.index.recovery.target.transport_address`**
:   type: keyword


**`elasticsearch.index.recovery.target.id`**
:   Target node id.

type: keyword


**`elasticsearch.index.recovery.target.host`**
:   Target node host address (could be IP address or hostname).

type: keyword


**`elasticsearch.index.recovery.target.name`**
:   Target node name.

type: keyword


**`elasticsearch.index.recovery.source.transport_address`**
:   type: keyword


**`elasticsearch.index.recovery.source.id`**
:   Source node id.

type: keyword


**`elasticsearch.index.recovery.source.host`**
:   Source node host address (could be IP address or hostname).

type: keyword


**`elasticsearch.index.recovery.source.name`**
:   Source node name.

type: keyword


**`elasticsearch.index.recovery.verify_index.check_index_time.ms`**
:   type: long


**`elasticsearch.index.recovery.verify_index.total_time.ms`**
:   type: long



## index.summary [_index_summary]

index

**`elasticsearch.index.summary.primaries.docs.count`**
:   Total number of documents in the index.

type: long


**`elasticsearch.index.summary.primaries.docs.deleted`**
:   Total number of deleted documents in the index.

type: long


**`elasticsearch.index.summary.primaries.store.size.bytes`**
:   Total size of the index in bytes.

type: long

format: bytes


**`elasticsearch.index.summary.primaries.store.total_data_set_size.bytes`**
:   Total size of the index in bytes including backing data for partially mounted indices.

type: long

format: bytes


**`elasticsearch.index.summary.primaries.segments.count`**
:   Total number of index segments.

type: long


**`elasticsearch.index.summary.primaries.segments.memory.bytes`**
:   Total number of memory used by the segments in bytes.

type: long

format: bytes


**`elasticsearch.index.summary.primaries.indexing.index.count`**
:   type: long


**`elasticsearch.index.summary.primaries.indexing.index.time.ms`**
:   type: long


**`elasticsearch.index.summary.primaries.search.query.count`**
:   type: long


**`elasticsearch.index.summary.primaries.search.query.time.ms`**
:   type: long


**`elasticsearch.index.summary.primaries.bulk.operations.count`**
:   type: long


**`elasticsearch.index.summary.primaries.bulk.size.bytes`**
:   type: long


**`elasticsearch.index.summary.primaries.bulk.time.count.ms`**
:   type: long


**`elasticsearch.index.summary.primaries.bulk.time.avg.ms`**
:   type: long


**`elasticsearch.index.summary.primaries.bulk.time.avg.bytes`**
:   type: long


**`elasticsearch.index.summary.total.docs.count`**
:   Total number of documents in the index.

type: long


**`elasticsearch.index.summary.total.docs.deleted`**
:   Total number of deleted documents in the index.

type: long


**`elasticsearch.index.summary.total.store.size.bytes`**
:   Total size of the index in bytes.

type: long

format: bytes


**`elasticsearch.index.summary.total.store.total_data_set_size.bytes`**
:   Total size of the index in bytes including backing data for partially mounted indices.

type: long

format: bytes


**`elasticsearch.index.summary.total.segments.count`**
:   Total number of index segments.

type: long


**`elasticsearch.index.summary.total.segments.memory.bytes`**
:   Total number of memory used by the segments in bytes.

type: long

format: bytes


**`elasticsearch.index.summary.total.indexing.index.count`**
:   type: long


**`elasticsearch.index.summary.total.indexing.is_throttled`**
:   type: boolean


**`elasticsearch.index.summary.total.indexing.throttle_time.ms`**
:   type: long


**`elasticsearch.index.summary.total.indexing.index.time.ms`**
:   type: long


**`elasticsearch.index.summary.total.search.query.count`**
:   type: long


**`elasticsearch.index.summary.total.search.query.time.ms`**
:   type: long


**`elasticsearch.index.summary.total.bulk.operations.count`**
:   type: long


**`elasticsearch.index.summary.total.bulk.size.bytes`**
:   type: long


**`elasticsearch.index.summary.total.bulk.time.avg.ms`**
:   type: long


**`elasticsearch.index.summary.total.bulk.time.avg.bytes`**
:   type: long



## ingest_pipeline [_ingest_pipeline]

Runtime metrics on ingest pipeline execution

**`elasticsearch.ingest_pipeline.name`**
:   Name / id of the ingest pipeline

type: wildcard



## total [_total]

Metrics on the total ingest pipeline execution, including all processors.

**`elasticsearch.ingest_pipeline.total.count`**
:   Number of documents processed by this pipeline

type: long


**`elasticsearch.ingest_pipeline.total.failed`**
:   Number of documented failed to process by this pipeline

type: long


**`elasticsearch.ingest_pipeline.total.time.total.ms`**
:   Total time spent processing documents through this pipeline, inclusive of other pipelines called

type: long


**`elasticsearch.ingest_pipeline.total.time.self.ms`**
:   Time spent processing documents through this pipeline, exclusive of other pipelines called

type: long


**`elasticsearch.ingest_pipeline.processor.type`**
:   The type of ingest processor

type: keyword


**`elasticsearch.ingest_pipeline.processor.type_tag`**
:   The type and the tag for this processor in the format "<type>:<tag>"

type: keyword


**`elasticsearch.ingest_pipeline.processor.order_index`**
:   The order this processor appears in the pipeline definition

type: long


**`elasticsearch.ingest_pipeline.processor.count`**
:   Number of documents processed by this processor

type: long


**`elasticsearch.ingest_pipeline.processor.failed`**
:   Number of documented failed to process by this processor

type: long


**`elasticsearch.ingest_pipeline.processor.time.total.ms`**
:   Total time spent processing documents through this processor

type: long



## ml.job [_ml_job]

ml

**`elasticsearch.ml.job.id`**
:   Unique ml job id.

type: keyword


**`elasticsearch.ml.job.state`**
:   Job state.

type: keyword


**`elasticsearch.ml.job.forecasts_stats.total`**
:   type: long


**`elasticsearch.ml.job.model_size.memory_status`**
:   type: keyword


**`elasticsearch.ml.job.data_counts.invalid_date_count`**
:   type: long


**`elasticsearch.ml.job.data_counts.processed_record_count`**
:   Processed data events.

type: long


**`elasticsearch.ml.job.data.invalid_date.count`**
:   The number of records with either a missing date field or a date that could not be parsed.

type: long



## node [_node_2]

node

**`elasticsearch.node.version`**
:   Node version.

type: keyword



## jvm [_jvm]

JVM Info.

**`elasticsearch.node.jvm.version`**
:   JVM version.

type: keyword


**`elasticsearch.node.jvm.memory.heap.init.bytes`**
:   Heap init used by the JVM in bytes.

type: long

format: bytes


**`elasticsearch.node.jvm.memory.heap.max.bytes`**
:   Heap max used by the JVM in bytes.

type: long

format: bytes


**`elasticsearch.node.jvm.memory.nonheap.init.bytes`**
:   Non-Heap init used by the JVM in bytes.

type: long

format: bytes


**`elasticsearch.node.jvm.memory.nonheap.max.bytes`**
:   Non-Heap max used by the JVM in bytes.

type: long

format: bytes


**`elasticsearch.node.process.mlockall`**
:   If process locked in memory.

type: boolean



## node.stats [_node_stats]

Statistics about each node in a Elasticsearch cluster

**`elasticsearch.node.stats.ingest.total.count`**
:   type: long


**`elasticsearch.node.stats.ingest.total.time_in_millis`**
:   type: long


**`elasticsearch.node.stats.ingest.total.current`**
:   type: long


**`elasticsearch.node.stats.ingest.total.failed`**
:   type: long


**`elasticsearch.node.stats.indices.bulk.avg_time.ms`**
:   type: long


**`elasticsearch.node.stats.indices.bulk.total_time.ms`**
:   type: long


**`elasticsearch.node.stats.indices.bulk.total_size.bytes`**
:   type: long


**`elasticsearch.node.stats.indices.bulk.avg_size.bytes`**
:   type: long


**`elasticsearch.node.stats.indices.bulk.operations.total.count`**
:   type: long


**`elasticsearch.node.stats.indices.docs.count`**
:   Total number of existing documents.

type: long


**`elasticsearch.node.stats.indices.docs.deleted`**
:   Total number of deleted documents.

type: long


**`elasticsearch.node.stats.indices.segments.count`**
:   Total number of segments.

type: long


**`elasticsearch.node.stats.indices.segments.memory.bytes`**
:   Total size of segments in bytes.

type: long

format: bytes


**`elasticsearch.node.stats.indices.store.size.bytes`**
:   Total size of all shards assigned to this node in bytes.

type: long


**`elasticsearch.node.stats.indices.store.total_data_set_size.bytes`**
:   Total size of shards in bytes assigned to this node including backing data for partially mounted indices.

type: long


**`elasticsearch.node.stats.indices.fielddata.evictions.count`**
:   type: long


**`elasticsearch.node.stats.indices.fielddata.memory.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indices.flush.total_time.ms`**
:   type: long


**`elasticsearch.node.stats.indices.flush.total.count`**
:   type: long


**`elasticsearch.node.stats.indices.get.time.ms`**
:   type: long


**`elasticsearch.node.stats.indices.get.total.count`**
:   type: long


**`elasticsearch.node.stats.indices.indexing.index_time.ms`**
:   type: long


**`elasticsearch.node.stats.indices.indexing.index_total.count`**
:   type: long


**`elasticsearch.node.stats.indices.indexing.throttle_time.ms`**
:   type: long


**`elasticsearch.node.stats.indices.merges.total_time.ms`**
:   type: long


**`elasticsearch.node.stats.indices.merges.total.count`**
:   type: long


**`elasticsearch.node.stats.indices.query_cache.memory.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indices.refresh.total_time.ms`**
:   type: long


**`elasticsearch.node.stats.indices.refresh.total.count`**
:   type: long


**`elasticsearch.node.stats.indices.request_cache.memory.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indices.search.query_time.ms`**
:   type: long


**`elasticsearch.node.stats.indices.search.query_total.count`**
:   type: long


**`elasticsearch.node.stats.indices.search.fetch_time.ms`**
:   type: long


**`elasticsearch.node.stats.indices.search.fetch_total.count`**
:   type: long


**`elasticsearch.node.stats.indices.shard_stats.total_count`**
:   type: long


**`elasticsearch.node.stats.indices.segments.doc_values.memory.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indices.segments.fixed_bit_set.memory.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indices.segments.index_writer.memory.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indices.segments.norms.memory.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indices.segments.points.memory.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indices.segments.stored_fields.memory.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indices.segments.term_vectors.memory.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indices.segments.terms.memory.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indices.segments.version_map.memory.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indices.translog.size.bytes`**
:   type: long


**`elasticsearch.node.stats.indices.translog.operations.count`**
:   type: long


**`elasticsearch.node.stats.jvm.mem.heap.max.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.jvm.mem.heap.used.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.jvm.mem.heap.used.pct`**
:   type: double

format: percent


**`elasticsearch.node.stats.jvm.threads.count`**
:   type: long


**`elasticsearch.node.stats.jvm.mem.pools.old.max.bytes`**
:   Max bytes.

type: long

format: bytes


**`elasticsearch.node.stats.jvm.mem.pools.old.peak.bytes`**
:   Peak bytes.

type: long

format: bytes


**`elasticsearch.node.stats.jvm.mem.pools.old.peak_max.bytes`**
:   Peak max bytes.

type: long

format: bytes


**`elasticsearch.node.stats.jvm.mem.pools.old.used.bytes`**
:   Used bytes.

type: long

format: bytes


**`elasticsearch.node.stats.jvm.mem.pools.young.max.bytes`**
:   Max bytes.

type: long

format: bytes


**`elasticsearch.node.stats.jvm.mem.pools.young.peak.bytes`**
:   Peak bytes.

type: long

format: bytes


**`elasticsearch.node.stats.jvm.mem.pools.young.peak_max.bytes`**
:   Peak max bytes.

type: long

format: bytes


**`elasticsearch.node.stats.jvm.mem.pools.young.used.bytes`**
:   Used bytes.

type: long

format: bytes


**`elasticsearch.node.stats.jvm.mem.pools.survivor.max.bytes`**
:   Max bytes.

type: long

format: bytes


**`elasticsearch.node.stats.jvm.mem.pools.survivor.peak.bytes`**
:   Peak bytes.

type: long

format: bytes


**`elasticsearch.node.stats.jvm.mem.pools.survivor.peak_max.bytes`**
:   Peak max bytes.

type: long

format: bytes


**`elasticsearch.node.stats.jvm.mem.pools.survivor.used.bytes`**
:   Used bytes.

type: long

format: bytes


**`elasticsearch.node.stats.jvm.gc.collectors.old.collection.count`**
:   type: long


**`elasticsearch.node.stats.jvm.gc.collectors.old.collection.ms`**
:   type: long


**`elasticsearch.node.stats.jvm.gc.collectors.young.collection.count`**
:   type: long


**`elasticsearch.node.stats.jvm.gc.collectors.young.collection.ms`**
:   type: long


**`elasticsearch.node.stats.fs.total.total_in_bytes`**
:   type: long


**`elasticsearch.node.stats.fs.total.available_in_bytes`**
:   type: long



## summary [_summary_4]

File system summary

**`elasticsearch.node.stats.fs.summary.total.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.fs.summary.free.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.fs.summary.available.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.fs.io_stats.total.operations.count`**
:   type: long


**`elasticsearch.node.stats.fs.io_stats.total.read.operations.count`**
:   type: long


**`elasticsearch.node.stats.fs.io_stats.total.write.operations.count`**
:   type: long


**`elasticsearch.node.stats.os.cpu.load_avg.1m`**
:   type: half_float


**`elasticsearch.node.stats.os.cgroup.cpuacct.usage.ns`**
:   type: long


**`elasticsearch.node.stats.os.cgroup.cpu.cfs.quota.us`**
:   type: long


**`elasticsearch.node.stats.os.cgroup.cpu.stat.elapsed_periods.count`**
:   type: long


**`elasticsearch.node.stats.os.cgroup.cpu.stat.times_throttled.count`**
:   type: long


**`elasticsearch.node.stats.os.cgroup.cpu.stat.time_throttled.ns`**
:   type: long


**`elasticsearch.node.stats.os.cgroup.memory.control_group`**
:   type: keyword


**`elasticsearch.node.stats.os.cgroup.memory.limit.bytes`**
:   type: keyword


**`elasticsearch.node.stats.os.cgroup.memory.usage.bytes`**
:   type: keyword


**`elasticsearch.node.stats.process.cpu.pct`**
:   type: double

format: percent


**`elasticsearch.node.stats.process.mem.total_virtual.bytes`**
:   type: long


**`elasticsearch.node.stats.process.open_file_descriptors`**
:   type: long


**`elasticsearch.node.stats.transport.rx.count`**
:   type: long


**`elasticsearch.node.stats.transport.rx.size.bytes`**
:   type: long


**`elasticsearch.node.stats.transport.tx.count`**
:   type: long


**`elasticsearch.node.stats.transport.tx.size.bytes`**
:   type: long


**`elasticsearch.node.stats.thread_pool.bulk.active.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.bulk.queue.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.bulk.rejected.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.esql_worker.active.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.esql_worker.queue.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.esql_worker.rejected.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.force_merge.active.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.force_merge.queue.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.force_merge.rejected.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.flush.active.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.flush.queue.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.flush.rejected.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.get.active.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.get.queue.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.get.rejected.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.index.active.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.index.queue.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.index.rejected.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.search.active.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.search.queue.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.search.rejected.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.search_worker.active.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.search_worker.queue.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.search_worker.rejected.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.snapshot.active.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.snapshot.queue.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.snapshot.rejected.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.system_read.active.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.system_read.queue.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.system_read.rejected.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.system_write.active.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.system_write.queue.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.system_write.rejected.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.write.active.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.write.queue.count`**
:   type: long


**`elasticsearch.node.stats.thread_pool.write.rejected.count`**
:   type: long


**`elasticsearch.node.stats.indexing_pressure.memory.current.combined_coordinating_and_primary.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indexing_pressure.memory.total.primary.rejections`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indexing_pressure.memory.total.primary.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indexing_pressure.memory.total.coordinating.rejections`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indexing_pressure.memory.total.coordinating.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indexing_pressure.memory.total.replica.rejections`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indexing_pressure.memory.total.replica.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indexing_pressure.memory.total.combined_coordinating_and_primary.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indexing_pressure.memory.current.coordinating.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indexing_pressure.memory.current.primary.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indexing_pressure.memory.current.replica.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indexing_pressure.memory.current.all.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indexing_pressure.memory.total.all.bytes`**
:   type: long

format: bytes


**`elasticsearch.node.stats.indexing_pressure.memory.limit_in_bytes`**
:   type: long

format: bytes



## cluster.pending_task [_cluster_pending_task]

`cluster.pending_task` contains a pending task description.

**`elasticsearch.cluster.pending_task.insert_order`**
:   Insert order

type: long


**`elasticsearch.cluster.pending_task.priority`**
:   Priority

type: long


**`elasticsearch.cluster.pending_task.source`**
:   Source. For example: put-mapping

type: keyword


**`elasticsearch.cluster.pending_task.time_in_queue.ms`**
:   Time in queue

type: long



## shard [_shard]

shard fields

**`elasticsearch.shard.primary`**
:   True if this is the primary shard.

type: boolean


**`elasticsearch.shard.number`**
:   The number of this shard.

type: long


**`elasticsearch.shard.state`**
:   The state of this shard.

type: keyword


**`elasticsearch.shard.relocating_node.name`**
:   The node the shard was relocated from.

type: keyword


**`elasticsearch.shard.relocating_node.id`**
:   The node the shard was relocated from. It has the exact same value than relocating_node.name for compatibility purposes.

type: keyword


**`elasticsearch.shard.source_node.name`**
:   type: keyword


**`elasticsearch.shard.source_node.uuid`**
:   type: keyword


