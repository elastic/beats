---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-docker.html
---

# Docker fields [exported-fields-docker]

Docker stats collected from Docker.


## docker [_docker_4]

Information and statistics about docker’s running containers.


## container [_container_3]

Docker container metrics.

**`docker.container.command`**
:   Command that was executed in the Docker container.

type: keyword


**`docker.container.created`**
:   Date when the container was created.

type: date


**`docker.container.status`**
:   Container status.

type: keyword


**`docker.container.ip_addresses`**
:   Container IP addresses.

type: ip



## size [_size]

Container size metrics.

**`docker.container.size.root_fs`**
:   Total size of all the files in the container.

type: long


**`docker.container.size.rw`**
:   Size of the files that have been created or changed since creation.

type: long


**`docker.container.tags`**
:   Image tags.

type: keyword



## cpu [_cpu_5]

Runtime CPU metrics.

**`docker.cpu.kernel.pct`**
:   Percentage of time in kernel space, expressed as a value between 0 and 1.

type: scaled_float

format: percent


**`docker.cpu.kernel.norm.pct`**
:   Percentage of time in kernel space normalized by the number of CPU cores, expressed as a value between 0 and 1.

type: scaled_float

format: percent


**`docker.cpu.kernel.ticks`**
:   CPU ticks in kernel space.

type: long


**`docker.cpu.system.pct`**
:   Percentage of total CPU time in the system, expressed as a value between 0 and 1.

type: scaled_float

format: percent


**`docker.cpu.system.norm.pct`**
:   Percentage of total CPU time in the system normalized by the number of CPU cores, expressed as a value between 0 and 1.

type: scaled_float

format: percent


**`docker.cpu.system.ticks`**
:   CPU system ticks.

type: long


**`docker.cpu.user.pct`**
:   Percentage of time in user space, expressed as a value between 0 and 1.

type: scaled_float

format: percent


**`docker.cpu.user.norm.pct`**
:   Percentage of time in user space normalized by the number of CPU cores, expressed as a value between 0 and 1.

type: scaled_float

format: percent


**`docker.cpu.user.ticks`**
:   CPU ticks in user space.

type: long


**`docker.cpu.total.pct`**
:   Total CPU usage.

type: scaled_float

format: percent


**`docker.cpu.total.norm.pct`**
:   Total CPU usage normalized by the number of CPU cores.

type: scaled_float

format: percent


**`docker.cpu.core.*.pct`**
:   Percentage of CPU time in this core, expressed as a value between 0 and 1.

type: object

format: percent


**`docker.cpu.core.*.norm.pct`**
:   Percentage of CPU time in this core normalized by the number of CPU cores, expressed as a value between 0 and 1.

type: object

format: percent


**`docker.cpu.core.*.ticks`**
:   Number of CPU ticks in this core.

type: object



## diskio [_diskio_3]

Disk I/O metrics.


## read [_read_5]

Accumulated reads during the life of the container

**`docker.diskio.read.ops`**
:   Number of reads during the life of the container

type: long


**`docker.diskio.read.bytes`**
:   Bytes read during the life of the container

type: long

format: bytes


**`docker.diskio.read.rate`**
:   Number of current reads per second

type: long


**`docker.diskio.read.service_time`**
:   Total time to service IO requests, in nanoseconds

type: long


**`docker.diskio.read.wait_time`**
:   Total time requests spent waiting in queues for service, in nanoseconds

type: long


**`docker.diskio.read.queued`**
:   Total number of queued requests

type: long



## write [_write_5]

Accumulated writes during the life of the container

**`docker.diskio.write.ops`**
:   Number of writes during the life of the container

type: long


**`docker.diskio.write.bytes`**
:   Bytes written during the life of the container

type: long

format: bytes


**`docker.diskio.write.rate`**
:   Number of current writes per second

type: long


**`docker.diskio.write.service_time`**
:   Total time to service IO requests, in nanoseconds

type: long


**`docker.diskio.write.wait_time`**
:   Total time requests spent waiting in queues for service, in nanoseconds

type: long


**`docker.diskio.write.queued`**
:   Total number of queued requests

type: long



## summary [_summary_3]

Accumulated reads and writes during the life of the container

**`docker.diskio.summary.ops`**
:   Number of I/O operations during the life of the container

type: long


**`docker.diskio.summary.bytes`**
:   Bytes read and written during the life of the container

type: long

format: bytes


**`docker.diskio.summary.rate`**
:   Number of current operations per second

type: long


**`docker.diskio.summary.service_time`**
:   Total time to service IO requests, in nanoseconds

type: long


**`docker.diskio.summary.wait_time`**
:   Total time requests spent waiting in queues for service, in nanoseconds

type: long


**`docker.diskio.summary.queued`**
:   Total number of queued requests

type: long



## event [_event]

Docker event

**`docker.event.status`**
:   Event status

type: keyword


**`docker.event.id`**
:   Event id when available

type: keyword


**`docker.event.from`**
:   Event source

type: keyword


**`docker.event.type`**
:   The type of object emitting the event

type: keyword


**`docker.event.action`**
:   The type of event

type: keyword



## actor [_actor]

Actor

**`docker.event.actor.id`**
:   The ID of the object emitting the event

type: keyword


**`docker.event.actor.attributes`**
:   Various key/value attributes of the object, depending on its type

type: object



## healthcheck [_healthcheck]

Docker healthcheck metrics. Healthcheck data will only be available from docker containers where the docker `HEALTHCHECK` instruction has been used to build the docker image.

**`docker.healthcheck.failingstreak`**
:   concurent failed check

type: integer


**`docker.healthcheck.status`**
:   Healthcheck status code

type: keyword



## event [_event_2]

event fields.

**`docker.healthcheck.event.end_date`**
:   Healthcheck end date

type: date


**`docker.healthcheck.event.start_date`**
:   Healthcheck start date

type: date


**`docker.healthcheck.event.output`**
:   Healthcheck output

type: keyword


**`docker.healthcheck.event.exit_code`**
:   Healthcheck status code

type: integer



## image [_image]

Docker image metrics.


## id [_id]

The image layers identifier.

**`docker.image.id.current`**
:   Unique image identifier given upon its creation.

type: keyword


**`docker.image.id.parent`**
:   Identifier of the image, if it exists, from which the current image directly descends.

type: keyword


**`docker.image.created`**
:   Date and time when the image was created.

type: date



## size [_size_2]

Image size layers.

**`docker.image.size.virtual`**
:   Size of the image.

type: long


**`docker.image.size.regular`**
:   Total size of the all cached images associated to the current image.

type: long


**`docker.image.labels`**
:   Image labels.

type: object


**`docker.image.tags`**
:   Image tags.

type: keyword



## info [_info_4]

Info metrics based on [https://docs.docker.com/engine/reference/api/docker_remote_api_v1.24/#/display-system-wide-information](https://docs.docker.com/engine/reference/api/docker_remote_api_v1.24/#/display-system-wide-information).


## containers [_containers]

Overall container stats.

**`docker.info.containers.paused`**
:   Total number of paused containers.

type: long


**`docker.info.containers.running`**
:   Total number of running containers.

type: long


**`docker.info.containers.stopped`**
:   Total number of stopped containers.

type: long


**`docker.info.containers.total`**
:   Total number of existing containers.

type: long


**`docker.info.id`**
:   Unique Docker host identifier.

type: keyword


**`docker.info.images`**
:   Total number of existing images.

type: long



## memory [_memory_5]

Memory metrics.

**`docker.memory.stats.*`**
:   Raw memory stats from the cgroups memory.stat interface

type: object



## commit [_commit_2]

Committed bytes on Windows

**`docker.memory.commit.total`**
:   Total bytes

type: long

format: bytes


**`docker.memory.commit.peak`**
:   Peak committed bytes on Windows

type: long

format: bytes


**`docker.memory.private_working_set.total`**
:   private working sets on Windows

type: long

format: bytes


**`docker.memory.fail.count`**
:   Fail counter.

type: scaled_float


**`docker.memory.limit`**
:   Memory limit.

type: long

format: bytes



## rss [_rss_2]

RSS memory stats.

**`docker.memory.rss.total`**
:   Total memory resident set size.

type: long

format: bytes


**`docker.memory.rss.pct`**
:   Memory resident set size percentage, expressed as a value between 0 and 1.

type: scaled_float

format: percent



## usage [_usage_13]

Usage memory stats.

**`docker.memory.usage.max`**
:   Max memory usage.

type: long

format: bytes


**`docker.memory.usage.pct`**
:   Memory usage percentage, expressed as a value between 0 and 1.

type: scaled_float

format: percent


**`docker.memory.usage.total`**
:   Total memory usage.

type: long

format: bytes



## network [_network_2]

Network metrics.

**`docker.network.interface`**
:   Network interface name.

type: keyword



## in [_in]

Incoming network stats per second.

**`docker.network.in.bytes`**
:   Incoming bytes per seconds.

type: long

format: bytes


**`docker.network.in.dropped`**
:   Dropped incoming packets per second.

type: scaled_float


**`docker.network.in.errors`**
:   Errors on incoming packets per second.

type: long


**`docker.network.in.packets`**
:   Incoming packets per second.

type: long



## out [_out]

Outgoing network stats per second.

**`docker.network.out.bytes`**
:   Outgoing bytes per second.

type: long

format: bytes


**`docker.network.out.dropped`**
:   Dropped outgoing packets per second.

type: scaled_float


**`docker.network.out.errors`**
:   Errors on outgoing packets per second.

type: long


**`docker.network.out.packets`**
:   Outgoing packets per second.

type: long



## inbound [_inbound]

Incoming network stats since the container started.

**`docker.network.inbound.bytes`**
:   Total number of incoming bytes.

type: long

format: bytes


**`docker.network.inbound.dropped`**
:   Total number of dropped incoming packets.

type: long


**`docker.network.inbound.errors`**
:   Total errors on incoming packets.

type: long


**`docker.network.inbound.packets`**
:   Total number of incoming packets.

type: long



## outbound [_outbound]

Outgoing network stats since the container started.

**`docker.network.outbound.bytes`**
:   Total number of outgoing bytes.

type: long

format: bytes


**`docker.network.outbound.dropped`**
:   Total number of dropped outgoing packets.

type: long


**`docker.network.outbound.errors`**
:   Total errors on outgoing packets.

type: long


**`docker.network.outbound.packets`**
:   Total number of outgoing packets.

type: long



## network_summary [_network_summary]

network_summary

**`docker.network_summary.ip.*`**
:   IP counters

type: object


**`docker.network_summary.tcp.*`**
:   TCP counters

type: object


**`docker.network_summary.udp.*`**
:   UDP counters

type: object


**`docker.network_summary.udp_lite.*`**
:   UDP Lite counters

type: object


**`docker.network_summary.icmp.*`**
:   ICMP counters

type: object


**`docker.network_summary.namespace.pid`**
:   The root PID of the container, corresponding to /proc/[pid]/net

type: long


**`docker.network_summary.namespace.id`**
:   The ID of the network namespace used by the container, corresponding to /proc/[pid]/ns/net

type: long


