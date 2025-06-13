---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-autoops_es.html
---

% This file is generated! See scripts/generate_fields_docs.py

# AutoOps ES fields [exported-fields-autoops_es]

AutoOps Elasticsearch module

## autoops_es [_autoops_es]



## cat_shards [_cat_shards]

cat shards information from the cluster

**`autoops_es.cat_shards.ip`**
:   Shard id

type: keyword


**`autoops_es.cat_shards.index`**
:   Shard index

type: keyword


**`autoops_es.cat_shards.shard`**
:   Shard number

type: keyword


**`autoops_es.cat_shards.prirep`**
:   Primary / Replica shard

type: keyword


**`autoops_es.cat_shards.state`**
:   State of the shard

type: keyword


**`autoops_es.cat_shards.docs`**
:   Docs count

type: long


**`autoops_es.cat_shards.store`**
:   Shard size in bytes

type: long


**`autoops_es.cat_shards.segments_count`**
:   Shard segments count

type: long


**`autoops_es.cat_shards.search_query_total`**
:   Shard search count

type: long


**`autoops_es.cat_shards.search_query_time`**
:   Shard search time in millis

type: long


**`autoops_es.cat_shards.indexing_index_total`**
:   Shard indexing total

type: long


**`autoops_es.cat_shards.indexing_index_time`**
:   Shard indexing time

type: long


**`autoops_es.cat_shards.indexing_index_failed`**
:   Shard indexing failed

type: long


**`autoops_es.cat_shards.merges_total`**
:   Shard merges total

type: long


**`autoops_es.cat_shards.merges_total_size`**
:   Shard merges size in bytes

type: long


**`autoops_es.cat_shards.merges_total_time`**
:   Shard merges time in millis

type: long


## cat_template [_cat_template]

tasks information from the cluster

**`autoops_es.cat_template.index`**
:   index name

type: keyword


**`autoops_es.cat_template.managed`**
:   indicate whether this index is ilm managed

type: boolean


**`autoops_es.cat_template.policy`**
:   policy name

type: keyword


**`autoops_es.cat_template.lifecycle_date_millis`**
:   lifecycle date in epoch millis

type: long


**`autoops_es.cat_template.lifecycle_date`**
:   lifecycle date

type: date


**`autoops_es.cat_template.phase`**
:   phase stage

type: keyword


**`autoops_es.cat_template.phase_time_millis`**
:   phase time in millis

type: long


**`autoops_es.cat_template.phase_time`**
:   phase time

type: date


**`autoops_es.cat_template.action`**
:   action name

type: keyword


## cluster_health [_cluster_health]

cluster health metrics

**`autoops_es.cluster_health.cluster_name`**
:   The cluster name

type: keyword


**`autoops_es.cluster_health.status`**
:   The cluster status

type: keyword


**`autoops_es.cluster_health.timed_out`**
:   Whether the call for status was timed out

type: keyword


**`autoops_es.cluster_health.number_of_nodes`**
:   Number of nodes on cluster

type: long


**`autoops_es.cluster_health.number_of_data_nodes`**
:   The number of data nodes

type: long


**`autoops_es.cluster_health.active_primary_shards`**
:   The number of active primary shards

type: long


**`autoops_es.cluster_health.active_shards`**
:   The number of active shards

type: long


**`autoops_es.cluster_health.relocating_shards`**
:   The number of relocating shards

type: long


**`autoops_es.cluster_health.initializing_shards`**
:   The number of initializing shards

type: long


**`autoops_es.cluster_health.unassigned_shards`**
:   The number of unassigned shards

type: long


**`autoops_es.cluster_health.delayed_unassigned_shards`**
:   The delayed unassigned shards

type: long


**`autoops_es.cluster_health.number_of_pending_tasks`**
:   The number of pending tasks

type: long


**`autoops_es.cluster_health.number_of_in_flight_fetch`**
:   The number of in flight_fetch

type: long


**`autoops_es.cluster_health.task_max_waiting_in_queue_millis`**
:   The task max waiting in queue millis

type: long


**`autoops_es.cluster_health.active_shards_percent_as_number`**
:   The active shards percent as number

type: long


## cluster_settings [_cluster_settings]

cluster_settings

## defaults [_defaults]

default settings

## discovery [_discovery]

discovery settings

## zen [_zen]

zen discovery settings

**`autoops_es.cluster_settings.defaults.discovery.zen.minimum_master_nodes`**
:   minimum_master_nodes

type: keyword


## cluster [_cluster]

cluster settings

**`autoops_es.cluster_settings.defaults.cluster.max_shards_per_node`**
:   max_shards_per_node

type: keyword


## routing [_routing]

routing settings

## allocation [_allocation]

allocation settings

## disk [_disk]

disk settings

## watermark [_watermark]

watermark settings

**`autoops_es.cluster_settings.defaults.cluster.routing.allocation.disk.watermark.low`**
:   low watermark settings

type: keyword


**`autoops_es.cluster_settings.defaults.cluster.routing.allocation.disk.watermark.high`**
:   high watermark settings

type: keyword


**`autoops_es.cluster_settings.defaults.cluster.routing.allocation.disk.watermark.flood_stage`**
:   flood_stage watermark settings

type: keyword


**`autoops_es.cluster_settings.defaults.cluster.routing.allocation.node_concurrent_outgoing_recoveries`**
:   node_concurrent_outgoing_recoveries

type: keyword


**`autoops_es.cluster_settings.defaults.cluster.routing.allocation.cluster_concurrent_rebalance`**
:   cluster_concurrent_rebalance

type: keyword


**`autoops_es.cluster_settings.defaults.cluster.routing.allocation.node_concurrent_recoveries`**
:   node_concurrent_recoveries

type: keyword


**`autoops_es.cluster_settings.defaults.cluster.routing.allocation.total_shards_per_node`**
:   total_shards_per_node

type: keyword


## blocks [_blocks]

blocks settings

**`autoops_es.cluster_settings.defaults.cluster.blocks.read_only`**
:   read_only settings

type: keyword


**`autoops_es.cluster_settings.defaults.cluster.blocks.create_index`**
:   create_index settings

type: keyword


**`autoops_es.cluster_settings.defaults.cluster.blocks.read_only_allow_delete`**
:   read_only_allow_delete settings

type: keyword


## bootstrap [_bootstrap]

bootstrap settings

**`autoops_es.cluster_settings.defaults.bootstrap.memory_lock`**
:   memory_lock settings

type: keyword


## search [_search]

search settings

**`autoops_es.cluster_settings.defaults.search.default_search_timeout`**
:   default_search_timeout

type: keyword


**`autoops_es.cluster_settings.defaults.search.max_buckets`**
:   max_buckets

type: keyword


## indices [_indices]

indices settings

## recovery [_recovery]

recovery settings

**`autoops_es.cluster_settings.defaults.indices.recovery.max_bytes_per_sec`**
:   max_bytes_per_sec settings

type: keyword


## breaker [_breaker]

breaker settings

## request [_request]

request breaker settings

**`autoops_es.cluster_settings.defaults.indices.breaker.request.limit`**
:   limit settings

type: keyword


## total [_total]

total breaker settings

**`autoops_es.cluster_settings.defaults.indices.breaker.total.limit`**
:   limit settings

type: keyword


## query [_query]

query settings

## query_string [_query_string]

query_string settings

**`autoops_es.cluster_settings.defaults.indices.query.query_string.allowLeadingWildcard`**
:   allowLeadingWildcard settings

type: keyword


## action [_action]

action settings

**`autoops_es.cluster_settings.defaults.action.destructive_requires_name`**
:   destructive_requires_name settings

type: keyword


## persistent [_persistent]

persistent settings

## discovery [_discovery]

discovery settings

## zen [_zen]

zen discovery settings

**`autoops_es.cluster_settings.persistent.discovery.zen.minimum_master_nodes`**
:   minimum_master_nodes

type: keyword


## cluster [_cluster]

cluster settings

**`autoops_es.cluster_settings.persistent.cluster.max_shards_per_node`**
:   max_shards_per_node

type: keyword


## routing [_routing]

routing settings

## allocation [_allocation]

allocation settings

## disk [_disk]

disk settings

## watermark [_watermark]

watermark settings

**`autoops_es.cluster_settings.persistent.cluster.routing.allocation.disk.watermark.low`**
:   low watermark settings

type: keyword


**`autoops_es.cluster_settings.persistent.cluster.routing.allocation.disk.watermark.high`**
:   high watermark settings

type: keyword


**`autoops_es.cluster_settings.persistent.cluster.routing.allocation.disk.watermark.flood_stage`**
:   flood_stage watermark settings

type: keyword


**`autoops_es.cluster_settings.persistent.cluster.routing.allocation.node_concurrent_outgoing_recoveries`**
:   node_concurrent_outgoing_recoveries

type: keyword


**`autoops_es.cluster_settings.persistent.cluster.routing.allocation.cluster_concurrent_rebalance`**
:   cluster_concurrent_rebalance

type: keyword


**`autoops_es.cluster_settings.persistent.cluster.routing.allocation.node_concurrent_recoveries`**
:   node_concurrent_recoveries

type: keyword


**`autoops_es.cluster_settings.persistent.cluster.routing.allocation.total_shards_per_node`**
:   total_shards_per_node

type: keyword


## blocks [_blocks]

blocks settings

**`autoops_es.cluster_settings.persistent.cluster.blocks.read_only`**
:   read_only settings

type: keyword


**`autoops_es.cluster_settings.persistent.cluster.blocks.create_index`**
:   create_index settings

type: keyword


**`autoops_es.cluster_settings.persistent.cluster.blocks.read_only_allow_delete`**
:   read_only_allow_delete settings

type: keyword


## bootstrap [_bootstrap]

bootstrap settings

**`autoops_es.cluster_settings.persistent.bootstrap.memory_lock`**
:   memory_lock settings

type: keyword


## search [_search]

search settings

**`autoops_es.cluster_settings.persistent.search.default_search_timeout`**
:   default_search_timeout

type: keyword


**`autoops_es.cluster_settings.persistent.search.max_buckets`**
:   max_buckets

type: keyword


## indices [_indices]

indices settings

## recovery [_recovery]

recovery settings

**`autoops_es.cluster_settings.persistent.indices.recovery.max_bytes_per_sec`**
:   max_bytes_per_sec settings

type: keyword


## breaker [_breaker]

breaker settings

## request [_request]

request breaker settings

**`autoops_es.cluster_settings.persistent.indices.breaker.request.limit`**
:   limit settings

type: keyword


## total [_total]

total breaker settings

**`autoops_es.cluster_settings.persistent.indices.breaker.total.limit`**
:   limit settings

type: keyword


## query [_query]

query settings

## query_string [_query_string]

query_string settings

**`autoops_es.cluster_settings.persistent.indices.query.query_string.allowLeadingWildcard`**
:   allowLeadingWildcard settings

type: keyword


## action [_action]

action settings

**`autoops_es.cluster_settings.persistent.action.destructive_requires_name`**
:   destructive_requires_name settings

type: keyword


## transient [_transient]

transient settings

## discovery [_discovery]

discovery settings

## zen [_zen]

zen discovery settings

**`autoops_es.cluster_settings.transient.discovery.zen.minimum_master_nodes`**
:   minimum_master_nodes

type: keyword


## cluster [_cluster]

cluster settings

**`autoops_es.cluster_settings.transient.cluster.max_shards_per_node`**
:   max_shards_per_node

type: keyword


## routing [_routing]

routing settings

## allocation [_allocation]

allocation settings

## disk [_disk]

disk settings

## watermark [_watermark]

watermark settings

**`autoops_es.cluster_settings.transient.cluster.routing.allocation.disk.watermark.low`**
:   low watermark settings

type: keyword


**`autoops_es.cluster_settings.transient.cluster.routing.allocation.disk.watermark.high`**
:   high watermark settings

type: keyword


**`autoops_es.cluster_settings.transient.cluster.routing.allocation.disk.watermark.flood_stage`**
:   flood_stage watermark settings

type: keyword


**`autoops_es.cluster_settings.transient.cluster.routing.allocation.node_concurrent_outgoing_recoveries`**
:   node_concurrent_outgoing_recoveries

type: keyword


**`autoops_es.cluster_settings.transient.cluster.routing.allocation.cluster_concurrent_rebalance`**
:   cluster_concurrent_rebalance

type: keyword


**`autoops_es.cluster_settings.transient.cluster.routing.allocation.node_concurrent_recoveries`**
:   node_concurrent_recoveries

type: keyword


**`autoops_es.cluster_settings.transient.cluster.routing.allocation.total_shards_per_node`**
:   total_shards_per_node

type: keyword


## blocks [_blocks]

blocks settings

**`autoops_es.cluster_settings.transient.cluster.blocks.read_only`**
:   read_only settings

type: keyword


**`autoops_es.cluster_settings.transient.cluster.blocks.create_index`**
:   create_index settings

type: keyword


**`autoops_es.cluster_settings.transient.cluster.blocks.read_only_allow_delete`**
:   read_only_allow_delete settings

type: keyword


## bootstrap [_bootstrap]

bootstrap settings

**`autoops_es.cluster_settings.transient.bootstrap.memory_lock`**
:   memory_lock settings

type: keyword


## search [_search]

search settings

**`autoops_es.cluster_settings.transient.search.default_search_timeout`**
:   default_search_timeout

type: keyword


**`autoops_es.cluster_settings.transient.search.max_buckets`**
:   max_buckets

type: keyword


## indices [_indices]

indices settings

## recovery [_recovery]

recovery settings

**`autoops_es.cluster_settings.transient.indices.recovery.max_bytes_per_sec`**
:   max_bytes_per_sec settings

type: keyword


## breaker [_breaker]

breaker settings

## request [_request]

request breaker settings

**`autoops_es.cluster_settings.transient.indices.breaker.request.limit`**
:   limit settings

type: keyword


## total [_total]

total breaker settings

**`autoops_es.cluster_settings.transient.indices.breaker.total.limit`**
:   limit settings

type: keyword


## query [_query]

query settings

## query_string [_query_string]

query_string settings

**`autoops_es.cluster_settings.transient.indices.query.query_string.allowLeadingWildcard`**
:   allowLeadingWildcard settings

type: keyword


## action [_action]

action settings

**`autoops_es.cluster_settings.transient.action.destructive_requires_name`**
:   destructive_requires_name settings

type: keyword


## component_template [_component_template]

component template information from the cluster

**`autoops_es.component_template.index`**
:   index name

type: keyword


**`autoops_es.component_template.managed`**
:   indicate whether this index is ilm managed

type: boolean


**`autoops_es.component_template.policy`**
:   policy name

type: keyword


**`autoops_es.component_template.lifecycle_date_millis`**
:   lifecycle date in epoch millis

type: long


**`autoops_es.component_template.lifecycle_date`**
:   lifecycle date

type: date


**`autoops_es.component_template.phase`**
:   phase stage

type: keyword


**`autoops_es.component_template.phase_time_millis`**
:   phase time in millis

type: long


**`autoops_es.component_template.phase_time`**
:   phase time

type: date


**`autoops_es.component_template.action`**
:   action name

type: keyword


## index_template [_index_template]

index templates from the cluster

**`autoops_es.index_template.index`**
:   index name

type: keyword


**`autoops_es.index_template.managed`**
:   indicate whether this index is ilm managed

type: boolean


**`autoops_es.index_template.policy`**
:   policy name

type: keyword


**`autoops_es.index_template.lifecycle_date_millis`**
:   lifecycle date in epoch millis

type: long


**`autoops_es.index_template.lifecycle_date`**
:   lifecycle date

type: date


**`autoops_es.index_template.phase`**
:   phase stage

type: keyword


**`autoops_es.index_template.phase_time_millis`**
:   phase time in millis

type: long


**`autoops_es.index_template.phase_time`**
:   phase time

type: date


**`autoops_es.index_template.action`**
:   action name

type: keyword


## node.stats [_node.stats]

node_stats

## indices [_indices]

Node indices stats

**`autoops_es.node.stats.indices.docs.count`**
:   Total number of existing documents.

type: long


**`autoops_es.node.stats.indices.docs.deleted`**
:   Total number of deleted documents.

type: long


**`autoops_es.node.stats.indices.segments.count`**
:   Total number of segments.

type: long


**`autoops_es.node.stats.indices.segments.memory.bytes`**
:   Total size of segments in bytes.

type: long

format: bytes


**`autoops_es.node.stats.indices.store.size.bytes`**
:   Total size of the store in bytes.

type: long


## jvm.mem.pools [_jvm.mem.pools]

JVM memory pool stats

## old [_old]

Old memory pool stats.

**`autoops_es.node.stats.jvm.mem.pools.old.max.bytes`**
:   Max bytes.

type: long

format: bytes


**`autoops_es.node.stats.jvm.mem.pools.old.peak.bytes`**
:   Peak bytes.

type: long

format: bytes


**`autoops_es.node.stats.jvm.mem.pools.old.peak_max.bytes`**
:   Peak max bytes.

type: long

format: bytes


**`autoops_es.node.stats.jvm.mem.pools.old.used.bytes`**
:   Used bytes.

type: long

format: bytes


## young [_young]

Young memory pool stats.

**`autoops_es.node.stats.jvm.mem.pools.young.max.bytes`**
:   Max bytes.

type: long

format: bytes


**`autoops_es.node.stats.jvm.mem.pools.young.peak.bytes`**
:   Peak bytes.

type: long

format: bytes


**`autoops_es.node.stats.jvm.mem.pools.young.peak_max.bytes`**
:   Peak max bytes.

type: long

format: bytes


**`autoops_es.node.stats.jvm.mem.pools.young.used.bytes`**
:   Used bytes.

type: long

format: bytes


## survivor [_survivor]

Survivor memory pool stats.

**`autoops_es.node.stats.jvm.mem.pools.survivor.max.bytes`**
:   Max bytes.

type: long

format: bytes


**`autoops_es.node.stats.jvm.mem.pools.survivor.peak.bytes`**
:   Peak bytes.

type: long

format: bytes


**`autoops_es.node.stats.jvm.mem.pools.survivor.peak_max.bytes`**
:   Peak max bytes.

type: long

format: bytes


**`autoops_es.node.stats.jvm.mem.pools.survivor.used.bytes`**
:   Used bytes.

type: long

format: bytes


## jvm.gc.collectors [_jvm.gc.collectors]

GC collector stats.

## old.collection [_old.collection]

Old collection gc.

**`autoops_es.node.stats.jvm.gc.collectors.old.collection.count`**
:   type: long


**`autoops_es.node.stats.jvm.gc.collectors.old.collection.ms`**
:   type: long


## young.collection [_young.collection]

Young collection gc.

**`autoops_es.node.stats.jvm.gc.collectors.young.collection.count`**
:   type: long


**`autoops_es.node.stats.jvm.gc.collectors.young.collection.ms`**
:   type: long


## fs.summary [_fs.summary]

File system summary

**`autoops_es.node.stats.fs.summary.total.bytes`**
:   type: long

format: bytes


**`autoops_es.node.stats.fs.summary.free.bytes`**
:   type: long

format: bytes


**`autoops_es.node.stats.fs.summary.available.bytes`**
:   type: long

format: bytes


## tasks_management [_tasks_management]

tasks information from cluster

**`autoops_es.tasks_management.taskId`**
:   task full id

type: keyword


**`autoops_es.tasks_management.id`**
:   task internal node id

type: integer


**`autoops_es.tasks_management.node`**
:   node id

type: keyword


**`autoops_es.tasks_management.taskType`**
:   task type

type: keyword


**`autoops_es.tasks_management.action`**
:   task action

type: keyword


**`autoops_es.tasks_management.startTimeInMillis`**
:   task start time in millis

type: long


**`autoops_es.tasks_management.runningTimeInNanos`**
:   task running time in nanos

type: long


**`autoops_es.tasks_management.parentTaskId`**
:   task parent id

type: keyword


