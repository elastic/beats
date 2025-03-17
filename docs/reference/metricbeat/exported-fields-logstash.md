---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-logstash.html
---

# Logstash fields [exported-fields-logstash]

Logstash module

**`logstash_stats.timestamp`**
:   type: alias

alias to: @timestamp


**`logstash_stats.jvm.mem.heap_used_in_bytes`**
:   type: alias

alias to: logstash.node.stats.jvm.mem.heap_used_in_bytes


**`logstash_stats.jvm.mem.heap_max_in_bytes`**
:   type: alias

alias to: logstash.node.stats.jvm.mem.heap_max_in_bytes


**`logstash_stats.jvm.uptime_in_millis`**
:   type: alias

alias to: logstash.node.stats.jvm.uptime_in_millis


**`logstash_stats.events.in`**
:   type: alias

alias to: logstash.node.stats.events.in


**`logstash_stats.events.out`**
:   type: alias

alias to: logstash.node.stats.events.out


**`logstash_stats.events.duration_in_millis`**
:   type: alias

alias to: logstash.node.stats.events.duration_in_millis


**`logstash_stats.logstash.uuid`**
:   type: alias

alias to: logstash.node.stats.logstash.uuid


**`logstash_stats.logstash.version`**
:   type: alias

alias to: logstash.node.stats.logstash.version


**`logstash_stats.pipelines`**
:   type: nested


**`logstash_stats.os.cpu.load_average.15m`**
:   type: alias

alias to: logstash.node.stats.os.cpu.load_average.15m


**`logstash_stats.os.cpu.load_average.1m`**
:   type: alias

alias to: logstash.node.stats.os.cpu.load_average.1m


**`logstash_stats.os.cpu.load_average.5m`**
:   type: alias

alias to: logstash.node.stats.os.cpu.load_average.5m


**`logstash_stats.os.cgroup.cpuacct.usage_nanos`**
:   type: alias

alias to: logstash.node.stats.os.cgroup.cpuacct.usage_nanos


**`logstash_stats.os.cgroup.cpu.cfs_quota_micros`**
:   type: alias

alias to: logstash.node.stats.os.cgroup.cpu.cfs_quota_micros


**`logstash_stats.os.cgroup.cpu.stat.number_of_elapsed_periods`**
:   type: alias

alias to: logstash.node.stats.os.cgroup.cpu.stat.number_of_elapsed_periods


**`logstash_stats.os.cgroup.cpu.stat.time_throttled_nanos`**
:   type: alias

alias to: logstash.node.stats.os.cgroup.cpu.stat.time_throttled_nanos


**`logstash_stats.os.cgroup.cpu.stat.number_of_times_throttled`**
:   type: alias

alias to: logstash.node.stats.os.cgroup.cpu.stat.number_of_times_throttled


**`logstash_stats.process.cpu.percent`**
:   type: alias

alias to: logstash.node.stats.process.cpu.percent


**`logstash_stats.queue.events_count`**
:   type: alias

alias to: logstash.node.stats.queue.events_count


**`logstash_state.pipeline.id`**
:   type: alias

alias to: logstash.node.state.pipeline.id


**`logstash_state.pipeline.hash`**
:   type: alias

alias to: logstash.node.state.pipeline.hash


**`logstash.elasticsearch.cluster.id`**
:   type: keyword



## node [_node_6]

node


## node [_node_7]

node_stats metrics.

**`logstash.node.id`**
:   type: keyword


**`logstash.node.state.pipeline.id`**
:   type: keyword


**`logstash.node.state.pipeline.hash`**
:   type: keyword


**`logstash.node.state.pipeline.ephemeral_id`**
:   type: keyword


**`logstash.node.state.pipeline.batch_size`**
:   type: long


**`logstash.node.state.pipeline.workers`**
:   type: long


**`logstash.node.state.pipeline.representation.hash`**
:   type: keyword


**`logstash.node.state.pipeline.representation.type`**
:   type: keyword


**`logstash.node.state.pipeline.representation.version`**
:   type: keyword


**`logstash.node.state.pipeline.representation.graph.edges`**
:   type: object


**`logstash.node.state.pipeline.representation.graph.vertices`**
:   type: object


**`logstash.node.host`**
:   Host name

type: alias

alias to: host.hostname


**`logstash.node.version`**
:   Logstash Version

type: alias

alias to: service.version



## jvm [_jvm_3]

JVM Info

**`logstash.node.jvm.version`**
:   Version

type: keyword


**`logstash.node.jvm.pid`**
:   Process ID

type: alias

alias to: process.pid


**`logstash.node.stats.timestamp`**
:   type: date


**`logstash.node.stats.jvm.uptime_in_millis`**
:   type: long


**`logstash.node.stats.jvm.mem.heap_used_in_bytes`**
:   type: long


**`logstash.node.stats.jvm.mem.heap_max_in_bytes`**
:   type: long



## events [_events_2]

Events stats

**`logstash.node.stats.events.in`**
:   Incoming events counter.

type: long


**`logstash.node.stats.events.out`**
:   Outgoing events counter.

type: long


**`logstash.node.stats.events.filtered`**
:   Filtered events counter.

type: long


**`logstash.node.stats.events.duration_in_millis`**
:   type: long


**`logstash.node.stats.logstash.uuid`**
:   type: keyword


**`logstash.node.stats.logstash.version`**
:   type: keyword


**`logstash.node.stats.os.cpu.load_average.15m`**
:   type: half_float


**`logstash.node.stats.os.cpu.load_average.1m`**
:   type: half_float


**`logstash.node.stats.os.cpu.load_average.5m`**
:   type: half_float


**`logstash.node.stats.os.cgroup.cpuacct.usage_nanos`**
:   type: long


**`logstash.node.stats.os.cgroup.cpu.cfs_quota_micros`**
:   type: long


**`logstash.node.stats.os.cgroup.cpu.stat.number_of_elapsed_periods`**
:   type: long


**`logstash.node.stats.os.cgroup.cpu.stat.time_throttled_nanos`**
:   type: long


**`logstash.node.stats.os.cgroup.cpu.stat.number_of_times_throttled`**
:   type: long


**`logstash.node.stats.process.cpu.percent`**
:   type: double


**`logstash.node.stats.pipelines`**
:   type: nested


**`logstash.node.stats.queue.events_count`**
:   type: long


