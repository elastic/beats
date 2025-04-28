---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-awsfargate.html
---

# AWS Fargate fields [exported-fields-awsfargate]

`awsfargate` module collects AWS fargate metrics from task metadata endpoint.

**`awsfargate.container.labels.com_amazonaws_ecs_cluster`**
:   ECS Cluster name

type: keyword


**`awsfargate.container.labels.com_amazonaws_ecs_container-name`**
:   ECS container name

type: keyword


**`awsfargate.container.labels.com_amazonaws_ecs_task-arn`**
:   ECS task ARN

type: keyword


**`awsfargate.container.labels.com_amazonaws_ecs_task-definition-family`**
:   ECS task definition family

type: keyword


**`awsfargate.container.labels.com_amazonaws_ecs_task-definition-version`**
:   ECS task definition version

type: keyword



## task_stats [_task_stats_2]

`task_stats` contains the metrics that were scraped from AWS fargate task stats ${ECS_CONTAINER_METADATA_URI_V4}/task/stats metadata endpoint.

**`awsfargate.task_stats.cluster_name`**
:   Cluster name (Pippero)

type: keyword


**`awsfargate.task_stats.task_name`**
:   ECS task name

type: keyword


**`awsfargate.task_stats.identifier`**
:   Container identifier across tasks and clusters, which equals to container.name + */* + container.id.

type: keyword


**`awsfargate.task_stats.task_desired_status`**
:   The desired status for the task from Amazon ECS.

type: keyword


**`awsfargate.task_stats.task_known_status`**
:   The known status for the task from Amazon ECS.

type: keyword


**`awsfargate.task_stats.memory_hard_limit`**
:   The Hard Memory Limit for the task from Amazon ECS.

type: scaled_float



## cpu [_cpu_3]

Runtime CPU metrics.

**`awsfargate.task_stats.cpu.kernel.pct`**
:   Percentage of time in kernel space, expressed as a value between 0 and 1.

type: scaled_float

format: percent


**`awsfargate.task_stats.cpu.kernel.norm.pct`**
:   Percentage of time in kernel space normalized by the number of CPU cores, expressed as a value between 0 and 1.

type: scaled_float

format: percent


**`awsfargate.task_stats.cpu.kernel.ticks`**
:   CPU ticks in kernel space.

type: long


**`awsfargate.task_stats.cpu.system.pct`**
:   Percentage of total CPU time in the system, expressed as a value between 0 and 1.

type: scaled_float

format: percent


**`awsfargate.task_stats.cpu.system.norm.pct`**
:   Percentage of total CPU time in the system normalized by the number of CPU cores, expressed as a value between 0 and 1.

type: scaled_float

format: percent


**`awsfargate.task_stats.cpu.system.ticks`**
:   CPU system ticks.

type: long


**`awsfargate.task_stats.cpu.user.pct`**
:   Percentage of time in user space, expressed as a value between 0 and 1.

type: scaled_float

format: percent


**`awsfargate.task_stats.cpu.user.norm.pct`**
:   Percentage of time in user space normalized by the number of CPU cores, expressed as a value between 0 and 1.

type: scaled_float

format: percent


**`awsfargate.task_stats.cpu.user.ticks`**
:   CPU ticks in user space.

type: long


**`awsfargate.task_stats.cpu.total.pct`**
:   Total CPU usage, expressed as a value between 0 and 1.

type: scaled_float

format: percent


**`awsfargate.task_stats.cpu.total.norm.pct`**
:   Total CPU usage normalized by the number of CPU cores, expressed as a value between 0 and 1.

type: scaled_float

format: percent



## diskio [_diskio_2]

Disk I/O metrics.


## read [_read_2]

Accumulated reads during the life of the container

**`awsfargate.task_stats.diskio.read.ops`**
:   Number of reads during the life of the container

type: long


**`awsfargate.task_stats.diskio.read.bytes`**
:   Bytes read during the life of the container

type: long

format: bytes


**`awsfargate.task_stats.diskio.read.rate`**
:   Number of current reads per second

type: long


**`awsfargate.task_stats.diskio.read.service_time`**
:   Total time to service IO requests, in nanoseconds

type: long


**`awsfargate.task_stats.diskio.read.wait_time`**
:   Total time requests spent waiting in queues for service, in nanoseconds

type: long


**`awsfargate.task_stats.diskio.read.queued`**
:   Total number of queued requests

type: long


**`awsfargate.task_stats.diskio.reads`**
:   :::{admonition} Deprecated in 6.4
    The `awsfargate.task_stats.diskio.reads` field was deprecated in 6.4.
    :::

Number of current reads per second

type: scaled_float



## write [_write_2]

Accumulated writes during the life of the container

**`awsfargate.task_stats.diskio.write.ops`**
:   Number of writes during the life of the container

type: long


**`awsfargate.task_stats.diskio.write.bytes`**
:   Bytes written during the life of the container

type: long

format: bytes


**`awsfargate.task_stats.diskio.write.rate`**
:   Number of current writes per second

type: long


**`awsfargate.task_stats.diskio.write.service_time`**
:   Total time to service IO requests, in nanoseconds

type: long


**`awsfargate.task_stats.diskio.write.wait_time`**
:   Total time requests spent waiting in queues for service, in nanoseconds

type: long


**`awsfargate.task_stats.diskio.write.queued`**
:   Total number of queued requests

type: long


**`awsfargate.task_stats.diskio.writes`**
:   :::{admonition} Deprecated in 6.4
    The `awsfargate.task_stats.diskio.writes` field was deprecated in 6.4.
    :::

Number of current writes per second

type: scaled_float



## summary [_summary]

Accumulated reads and writes during the life of the container

**`awsfargate.task_stats.diskio.summary.ops`**
:   Number of I/O operations during the life of the container

type: long


**`awsfargate.task_stats.diskio.summary.bytes`**
:   Bytes read and written during the life of the container

type: long

format: bytes


**`awsfargate.task_stats.diskio.summary.rate`**
:   Number of current operations per second

type: long


**`awsfargate.task_stats.diskio.summary.service_time`**
:   Total time to service IO requests, in nanoseconds

type: long


**`awsfargate.task_stats.diskio.summary.wait_time`**
:   Total time requests spent waiting in queues for service, in nanoseconds

type: long


**`awsfargate.task_stats.diskio.summary.queued`**
:   Total number of queued requests

type: long


**`awsfargate.task_stats.diskio.total`**
:   :::{admonition} Deprecated in 6.4
    The `aawsfargate.task_stats.diskio.total` field was deprecated in 6.4.
    :::

Number of reads and writes per second

type: scaled_float



## memory [_memory_3]

Memory metrics.

**`awsfargate.task_stats.memory.stats`**
:   Raw memory stats from the cgroups memory.stat interface

type: object



## commit [_commit]

Committed bytes on Windows

**`awsfargate.task_stats.memory.commit.total`**
:   Total bytes

type: long

format: bytes


**`awsfargate.task_stats.memory.commit.peak`**
:   Peak committed bytes on Windows

type: long

format: bytes


**`awsfargate.task_stats.memory.private_working_set.total`**
:   private working sets on Windows

type: long

format: bytes


**`awsfargate.task_stats.memory.fail.count`**
:   Fail counter.

type: scaled_float


**`awsfargate.task_stats.memory.limit`**
:   Memory limit.

type: long

format: bytes



## rss [_rss]

RSS memory stats.

**`awsfargate.task_stats.memory.rss.total`**
:   Total memory resident set size.

type: long

format: bytes


**`awsfargate.task_stats.memory.rss.pct`**
:   Memory resident set size percentage, expressed as a value between 0 and 1.

type: scaled_float

format: percent



## usage [_usage_11]

Usage memory stats.

**`awsfargate.task_stats.memory.usage.max`**
:   Max memory usage.

type: long

format: bytes


**`awsfargate.task_stats.memory.usage.total`**
:   Total memory usage.

type: long

format: bytes


**`awsfargate.task_stats.network.*.inbound.bytes`**
:   Total number of incoming bytes.

type: long

format: bytes


**`awsfargate.task_stats.network.*.inbound.dropped`**
:   Total number of dropped incoming packets.

type: long


**`awsfargate.task_stats.network.*.inbound.errors`**
:   Total errors on incoming packets.

type: long


**`awsfargate.task_stats.network.*.inbound.packets`**
:   Total number of incoming packets.

type: long


**`awsfargate.task_stats.network.*.outbound.bytes`**
:   Total number of incoming bytes.

type: long

format: bytes


**`awsfargate.task_stats.network.*.outbound.dropped`**
:   Total number of dropped incoming packets.

type: long


**`awsfargate.task_stats.network.*.outbound.errors`**
:   Total errors on incoming packets.

type: long


**`awsfargate.task_stats.network.*.outbound.packets`**
:   Total number of incoming packets.

type: long


