---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-kibana.html
---

# Kibana fields [exported-fields-kibana]

Kibana module

**`kibana_stats.timestamp`**
:   type: alias

alias to: @timestamp


**`kibana_stats.kibana.response_time.max`**
:   type: alias

alias to: kibana.stats.response_time.max.ms


**`kibana_stats.kibana.status`**
:   type: alias

alias to: kibana.stats.kibana.status


**`kibana_stats.os.memory.free_in_bytes`**
:   type: alias

alias to: kibana.stats.os.memory.free_in_bytes


**`kibana_stats.process.uptime_in_millis`**
:   type: alias

alias to: kibana.stats.process.uptime.ms


**`kibana_stats.process.memory.heap.size_limit`**
:   type: alias

alias to: kibana.stats.process.memory.heap.size_limit.bytes


**`kibana_stats.concurrent_connections`**
:   type: alias

alias to: kibana.stats.concurrent_connections


**`kibana_stats.process.memory.resident_set_size_in_bytes`**
:   type: alias

alias to: kibana.stats.process.memory.resident_set_size.bytes


**`kibana_stats.os.load.1m`**
:   type: alias

alias to: kibana.stats.os.load.1m


**`kibana_stats.os.load.5m`**
:   type: alias

alias to: kibana.stats.os.load.5m


**`kibana_stats.os.load.15m`**
:   type: alias

alias to: kibana.stats.os.load.15m


**`kibana_stats.process.event_loop_delay`**
:   type: alias

alias to: kibana.stats.process.event_loop_delay.ms


**`kibana_stats.process.event_loop_utilization.active`**
:   type: alias

alias to: kibana.stats.process.event_loop_utilization.active


**`kibana_stats.process.event_loop_utilization.idle`**
:   type: alias

alias to: kibana.stats.process.event_loop_utilization.idle


**`kibana_stats.process.event_loop_utilization.utilization`**
:   type: alias

alias to: kibana.stats.process.event_loop_utilization.utilization


**`kibana_stats.requests.total`**
:   type: alias

alias to: kibana.stats.request.total


**`kibana_stats.requests.disconnects`**
:   type: alias

alias to: kibana.stats.request.disconnects


**`kibana_stats.response_times.max`**
:   type: alias

alias to: kibana.stats.response_time.max.ms


**`kibana_stats.response_times.average`**
:   type: alias

alias to: kibana.stats.response_time.avg.ms


**`kibana_stats.kibana.uuid`**
:   type: alias

alias to: service.id


**`kibana.elasticsearch.cluster.id`**
:   type: keyword



## cluster_actions [_cluster_actions]

Kibana cluster actions metrics.

**`kibana.cluster_actions.kibana.status`**
:   type: keyword


**`kibana.cluster_actions.overdue.count`**
:   type: long


**`kibana.cluster_actions.overdue.delay.p50`**
:   type: float


**`kibana.cluster_actions.overdue.delay.p99`**
:   type: float



## cluster_rules [_cluster_rules]

Kibana cluster rule metrics.

**`kibana.cluster_rules.kibana.status`**
:   type: keyword


**`kibana.cluster_rules.overdue.count`**
:   type: long


**`kibana.cluster_rules.overdue.delay.p50`**
:   type: float


**`kibana.cluster_rules.overdue.delay.p99`**
:   type: float



## node_actions [_node_actions]

Kibana node actions metrics.

**`kibana.node_actions.kibana.status`**
:   type: keyword


**`kibana.node_actions.failures`**
:   type: long


**`kibana.node_actions.executions`**
:   type: long


**`kibana.node_actions.timeouts`**
:   type: long



## node_rules [_node_rules]

Kibana node rule metrics.

**`kibana.node_rules.kibana.status`**
:   type: keyword


**`kibana.node_rules.failures`**
:   type: long


**`kibana.node_rules.executions`**
:   type: long


**`kibana.node_rules.timeouts`**
:   type: long



## settings [_settings_2]

Kibana stats and run-time metrics.

**`kibana.settings.uuid`**
:   Kibana instance UUID

type: keyword


**`kibana.settings.name`**
:   Kibana instance name

type: keyword


**`kibana.settings.index`**
:   Name of Kibana’s internal index

type: keyword


**`kibana.settings.host`**
:   Kibana instance hostname

type: keyword


**`kibana.settings.transport_address`**
:   Kibana server’s hostname and port

type: keyword


**`kibana.settings.version`**
:   Kibana version

type: keyword


**`kibana.settings.snapshot`**
:   Whether the Kibana build is a snapshot build

type: boolean


**`kibana.settings.status`**
:   Kibana instance’s health status

type: keyword


**`kibana.settings.locale`**
:   type: keyword


**`kibana.settings.port`**
:   type: integer



## stats [_stats_5]

Kibana stats and run-time metrics.

**`kibana.stats.kibana.status`**
:   type: keyword


**`kibana.stats.usage.index`**
:   type: keyword


**`kibana.stats.uuid`**
:   Kibana instance UUID

type: alias

alias to: service.id


**`kibana.stats.name`**
:   Kibana instance name

type: keyword


**`kibana.stats.index`**
:   Name of Kibana’s internal index

type: keyword


**`kibana.stats.host.name`**
:   Kibana instance hostname

type: keyword


**`kibana.stats.transport_address`**
:   Kibana server’s hostname and port

type: alias

alias to: service.address


**`kibana.stats.version`**
:   Kibana version

type: alias

alias to: service.version


**`kibana.stats.snapshot`**
:   Whether the Kibana build is a snapshot build

type: boolean


**`kibana.stats.status`**
:   Kibana instance’s health status

type: keyword


**`kibana.stats.os.distro`**
:   type: keyword


**`kibana.stats.os.distroRelease`**
:   type: keyword


**`kibana.stats.os.platform`**
:   type: keyword


**`kibana.stats.os.platformRelease`**
:   type: keyword


**`kibana.stats.os.memory.free_in_bytes`**
:   type: long


**`kibana.stats.os.memory.total_in_bytes`**
:   type: long


**`kibana.stats.os.memory.used_in_bytes`**
:   type: long


**`kibana.stats.os.cpuacct.control_group`**
:   type: keyword


**`kibana.stats.os.cpuacct.usage_nanos`**
:   type: long


**`kibana.stats.os.cgroup_memory.current_in_bytes`**
:   type: long


**`kibana.stats.os.cgroup_memory.swap_current_in_bytes`**
:   type: long


**`kibana.stats.os.load.1m`**
:   type: half_float


**`kibana.stats.os.load.5m`**
:   type: half_float


**`kibana.stats.os.load.15m`**
:   type: half_float


**`kibana.stats.concurrent_connections`**
:   Number of client connections made to the server. Note that browsers can send multiple simultaneous connections to request multiple server assets at once, and they can re-use established connections.

type: long



## process [_process_6]

Process metrics

**`kibana.stats.process.memory.resident_set_size.bytes`**
:   type: long


**`kibana.stats.process.memory.array_buffers.bytes`**
:   type: long


**`kibana.stats.process.memory.external.bytes`**
:   type: long


**`kibana.stats.process.uptime.ms`**
:   type: long


**`kibana.stats.process.event_loop_delay.ms`**
:   Event loop delay in milliseconds

type: scaled_float



## event_loop_utilization [_event_loop_utilization]

The ratio of time the event loop is not idling in the event provider to the total time the event loop is running.

**`kibana.stats.process.event_loop_utilization.active`**
:   Duration of time event loop has been active since last measurement.

type: scaled_float


**`kibana.stats.process.event_loop_utilization.idle`**
:   Duration of time event loop has been idle since last measurement.

type: scaled_float


**`kibana.stats.process.event_loop_utilization.utilization`**
:   Computed utilization value representing ratio of active to idle time since last measurement.

type: scaled_float



## memory.heap [_memory_heap]

Process heap metrics

**`kibana.stats.process.memory.heap.total.bytes`**
:   Total heap allocated to process in bytes

type: long

format: bytes


**`kibana.stats.process.memory.heap.used.bytes`**
:   Heap used by process in bytes

type: long

format: bytes


**`kibana.stats.process.memory.heap.size_limit.bytes`**
:   Max. old space size allocated to Node.js process, in bytes

type: long

format: bytes


**`kibana.stats.process.memory.heap.uptime.ms`**
:   Uptime of process in milliseconds

type: long



## request [_request_2]

Request count metrics

**`kibana.stats.request.disconnects`**
:   Number of requests that were disconnected

type: long


**`kibana.stats.request.total`**
:   Total number of requests

type: long



## response_time [_response_time]

Response times metrics

**`kibana.stats.response_time.avg.ms`**
:   Average response time in milliseconds

type: long


**`kibana.stats.response_time.max.ms`**
:   Maximum response time in milliseconds

type: long



## elasticsearch_client [_elasticsearch_client]

Elasticsearch Client’s stats

**`kibana.stats.elasticsearch_client.total_active_sockets`**
:   Total number of active sockets

type: integer


**`kibana.stats.elasticsearch_client.total_idle_sockets`**
:   Total number of idle sockets

type: integer


**`kibana.stats.elasticsearch_client.total_queued_requests`**
:   Total number of queued requests

type: integer



## status [_status_2]

Status fields

**`kibana.status.name`**
:   Kibana instance name.

type: keyword


**`kibana.status.uuid`**
:   Kibana instance uuid.

type: alias

alias to: service.id


**`kibana.status.version.number`**
:   Kibana version number.

type: alias

alias to: service.version


**`kibana.status.status.overall.state`**
:   Kibana overall state (v7 format).

type: keyword


**`kibana.status.status.overall.level`**
:   Kibana overall level (v8 format).

type: keyword


**`kibana.status.status.overall.summary`**
:   Kibana overall state in a human-readable format.

type: text


**`kibana.status.status.core.elasticsearch.level`**
:   Kibana Elasticsearch client’s status

type: keyword


**`kibana.status.status.core.elasticsearch.summary`**
:   Kibana Elasticsearch client’s status in a human-readable format.

type: text


**`kibana.status.status.core.savedObjects.level`**
:   Kibana Saved Objects client’s status

type: keyword


**`kibana.status.status.core.savedObjects.summary`**
:   Kibana Saved Objects client’s status in a human-readable format.

type: text



## metrics [_metrics_8]

Metrics fields

**`kibana.status.metrics.concurrent_connections`**
:   Current concurrent connections.

type: long



## requests [_requests]

Request statistics.

**`kibana.status.metrics.requests.disconnects`**
:   Total number of disconnected connections.

type: long


**`kibana.status.metrics.requests.total`**
:   Total number of connections.

type: long


