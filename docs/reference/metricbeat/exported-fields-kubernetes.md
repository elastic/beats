---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-kubernetes.html
---

# Kubernetes fields [exported-fields-kubernetes]

Kubernetes metrics


## kubernetes [_kubernetes_3]

Information and statistics of pods managed by kubernetes.


## apiserver [_apiserver_2]

Kubernetes API server metrics

**`kubernetes.apiserver.major.version`**
:   API Server major version.

type: keyword


**`kubernetes.apiserver.minor.version`**
:   API Server minor version.

type: keyword


**`kubernetes.apiserver.request.resource`**
:   Requested resource

type: keyword


**`kubernetes.apiserver.request.subresource`**
:   Requested subresource

type: keyword


**`kubernetes.apiserver.request.scope`**
:   Request scope (cluster, namespace, resource)

type: keyword


**`kubernetes.apiserver.request.verb`**
:   HTTP verb

type: keyword


**`kubernetes.apiserver.request.code`**
:   HTTP code

type: keyword


**`kubernetes.apiserver.request.content_type`**
:   Request HTTP content type

type: keyword


**`kubernetes.apiserver.request.dry_run`**
:   Wether the request uses dry run

type: keyword


**`kubernetes.apiserver.request.kind`**
:   Kind of request

type: keyword


**`kubernetes.apiserver.request.component`**
:   Component handling the request

type: keyword


**`kubernetes.apiserver.request.group`**
:   API group for the resource

type: keyword


**`kubernetes.apiserver.request.version`**
:   version for the group

type: keyword


**`kubernetes.apiserver.request.handler`**
:   Request handler

type: keyword


**`kubernetes.apiserver.request.method`**
:   HTTP method

type: keyword


**`kubernetes.apiserver.request.host`**
:   Request host

type: keyword


**`kubernetes.apiserver.process.cpu.sec`**
:   CPU seconds

type: double


**`kubernetes.apiserver.process.memory.resident.bytes`**
:   Bytes in resident memory

type: long

format: bytes


**`kubernetes.apiserver.process.memory.virtual.bytes`**
:   Bytes in virtual memory

type: long

format: bytes


**`kubernetes.apiserver.process.fds.open.count`**
:   Number of open file descriptors

type: long


**`kubernetes.apiserver.process.started.sec`**
:   Seconds since the process started

type: double


**`kubernetes.apiserver.watch.events.size.bytes.bucket.*`**
:   Watch event size distribution in bytes

type: object


**`kubernetes.apiserver.watch.events.size.bytes.sum`**
:   Sum of watch events sizes in bytes

type: long

format: bytes


**`kubernetes.apiserver.watch.events.size.bytes.count`**
:   Number of watch events

type: long


**`kubernetes.apiserver.watch.events.kind`**
:   Resource kind of the watch event

type: keyword


**`kubernetes.apiserver.response.size.bytes.bucket.*`**
:   Response size distribution in bytes for each group, version, verb, resource, subresource, scope and component.

type: object


**`kubernetes.apiserver.response.size.bytes.sum`**
:   Sum of responses sizes in bytes

type: long

format: bytes


**`kubernetes.apiserver.response.size.bytes.count`**
:   Number of responses to requests

type: long


**`kubernetes.apiserver.client.request.count`**
:   Number of requests as client

type: long


**`kubernetes.apiserver.request.count`**
:   Number of requests

type: long


**`kubernetes.apiserver.request.duration.us.sum`**
:   Request duration, sum in microseconds

type: long


**`kubernetes.apiserver.request.duration.us.count`**
:   Request duration, number of operations

type: long


**`kubernetes.apiserver.request.duration.us.bucket.*`**
:   Response latency distribution, histogram buckets

type: object


**`kubernetes.apiserver.request.current.count`**
:   Inflight requests

type: long


**`kubernetes.apiserver.request.longrunning.count`**
:   Number of requests active long running requests

type: long


**`kubernetes.apiserver.etcd.object.count`**
:   Number of kubernetes objects at etcd

type: long


**`kubernetes.apiserver.audit.event.count`**
:   Number of audit events

type: long


**`kubernetes.apiserver.audit.rejected.count`**
:   Number of audit rejected events

type: long



## container [_container_5]

kubernetes container metrics

**`kubernetes.container.start_time`**
:   Start time

type: date



## cpu [_cpu_6]

CPU usage metrics

**`kubernetes.container.cpu.usage.core.ns`**
:   Container CPU Core usage nanoseconds

type: double


**`kubernetes.container.cpu.usage.nanocores`**
:   CPU used nanocores

type: double


**`kubernetes.container.cpu.usage.node.pct`**
:   CPU usage as a percentage of the total node allocatable CPU

type: scaled_float

format: percent


**`kubernetes.container.cpu.usage.limit.pct`**
:   CPU usage as a percentage of the defined limit for the container (or total node allocatable CPU if unlimited). If the container CPU limits are missing and the `node` and `state_node` metricsets are both disabled on that node, this metric will be missing entirely.

type: scaled_float

format: percent



## logs [_logs_2]

Logs info

**`kubernetes.container.logs.available.bytes`**
:   Logs available capacity in bytes

type: double

format: bytes


**`kubernetes.container.logs.capacity.bytes`**
:   Logs total capacity in bytes

type: double

format: bytes


**`kubernetes.container.logs.used.bytes`**
:   Logs used capacity in bytes

type: double

format: bytes


**`kubernetes.container.logs.inodes.count`**
:   Total available inodes

type: double


**`kubernetes.container.logs.inodes.free`**
:   Total free inodes

type: double


**`kubernetes.container.logs.inodes.used`**
:   Total used inodes

type: double


**`kubernetes.container.memory.available.bytes`**
:   Total available memory

type: double

format: bytes


**`kubernetes.container.memory.usage.bytes`**
:   Total memory usage

type: double

format: bytes


**`kubernetes.container.memory.usage.node.pct`**
:   Memory usage as a percentage of the total node allocatable memory

type: scaled_float

format: percent


**`kubernetes.container.memory.usage.limit.pct`**
:   Memory usage as a percentage of the defined limit for the container (or total node allocatable memory if unlimited). If the container Memory limits are missing and the `node` and `state_node` metricsets are both disabled on that node, this metric will be missing entirely.

type: scaled_float

format: percent


**`kubernetes.container.memory.rss.bytes`**
:   RSS memory usage

type: double

format: bytes


**`kubernetes.container.memory.workingset.bytes`**
:   Working set memory usage

type: double

format: bytes


**`kubernetes.container.memory.workingset.limit.pct`**
:   Working set memory usage as a percentage of the defined limit for the container (or total node allocatable memory if unlimited)

type: scaled_float

format: percent


**`kubernetes.container.memory.pagefaults`**
:   Number of page faults

type: double


**`kubernetes.container.memory.majorpagefaults`**
:   Number of major page faults

type: double


**`kubernetes.container.rootfs.capacity.bytes`**
:   Root filesystem total capacity in bytes

type: double

format: bytes


**`kubernetes.container.rootfs.available.bytes`**
:   Root filesystem total available in bytes

type: double

format: bytes


**`kubernetes.container.rootfs.used.bytes`**
:   Root filesystem total used in bytes

type: double

format: bytes


**`kubernetes.container.rootfs.inodes.used`**
:   Used inodes

type: double



## controllermanager [_controllermanager]

Controller manager metrics

**`kubernetes.controllermanager.verb`**
:   HTTP verb

type: keyword


**`kubernetes.controllermanager.code`**
:   HTTP code

type: keyword


**`kubernetes.controllermanager.method`**
:   HTTP method

type: keyword


**`kubernetes.controllermanager.host`**
:   HTTP host

type: keyword


**`kubernetes.controllermanager.name`**
:   Name for the resource

type: keyword


**`kubernetes.controllermanager.zone`**
:   Infrastructure zone

type: keyword


**`kubernetes.controllermanager.process.cpu.sec`**
:   Total user and system CPU time spent in seconds

type: double


**`kubernetes.controllermanager.process.memory.resident.bytes`**
:   Bytes in resident memory

type: long

format: bytes


**`kubernetes.controllermanager.process.memory.virtual.bytes`**
:   Bytes in virtual memory

type: long

format: bytes


**`kubernetes.controllermanager.process.fds.open.count`**
:   Number of open file descriptors

type: long


**`kubernetes.controllermanager.process.fds.max.count`**
:   Limit for open file descriptors

type: long


**`kubernetes.controllermanager.process.started.sec`**
:   Start time of the process since unix epoch in seconds

type: double


**`kubernetes.controllermanager.client.request.count`**
:   Number of HTTP requests to API server, broken down by status code, method and host

type: long


**`kubernetes.controllermanager.client.request.duration.us.sum`**
:   Sum of requests latency in microseconds, broken down by verb and host

type: long


**`kubernetes.controllermanager.client.request.duration.us.count`**
:   Number of request duration operations to API server, broken down by verb and host

type: long


**`kubernetes.controllermanager.client.request.duration.us.bucket.*`**
:   Requests latency distribution in histogram buckets, broken down by verb and host

type: object


**`kubernetes.controllermanager.client.request.size.bytes.sum`**
:   Requests size sum in bytes, broken down by verb and host

type: long

format: bytes


**`kubernetes.controllermanager.client.request.size.bytes.count`**
:   Number of requests, broken down by verb and host

type: long


**`kubernetes.controllermanager.client.request.size.bytes.bucket.*`**
:   Requests size distribution in histogram buckets, broken down by verb and host

type: object


**`kubernetes.controllermanager.client.response.size.bytes.count`**
:   Number of responses, broken down by verb and host

type: long


**`kubernetes.controllermanager.client.response.size.bytes.sum`**
:   Responses size sum in bytes, broken down by verb and host

type: long

format: bytes


**`kubernetes.controllermanager.client.response.size.bytes.bucket.*`**
:   Responses size distribution in histogram buckets, broken down by verb and host

type: object


**`kubernetes.controllermanager.workqueue.longestrunning.sec`**
:   How many seconds has the longest running processor been running, broken down by workqueue name

type: double


**`kubernetes.controllermanager.workqueue.unfinished.sec`**
:   How many seconds of work has done that is in progress and hasn’t been considered in the longest running processor, broken down by workqueue name

type: double


**`kubernetes.controllermanager.workqueue.adds.count`**
:   Workqueue add count, broken down by workqueue name

type: long


**`kubernetes.controllermanager.workqueue.depth.count`**
:   Workqueue current depth, broken down by workqueue name

type: long


**`kubernetes.controllermanager.workqueue.retries.count`**
:   Workqueue number of retries, broken down by workqueue name

type: long


**`kubernetes.controllermanager.node.collector.eviction.count`**
:   Number of node evictions, broken down by zone

type: long


**`kubernetes.controllermanager.node.collector.unhealthy.count`**
:   Number of unhealthy nodes, broken down by zone

type: long


**`kubernetes.controllermanager.node.collector.count`**
:   Number of nodes, broken down by zone

type: long


**`kubernetes.controllermanager.node.collector.health.pct`**
:   Percentage of healthy nodes, broken down by zone

type: long


**`kubernetes.controllermanager.leader.is_master`**
:   Whether the controller manager instance is leader

type: boolean



## event [_event_4]

The Kubernetes events metricset collects events that are generated by objects running inside of Kubernetes

**`kubernetes.event.count`**
:   Count field records the number of times the particular event has occurred

type: long


**`kubernetes.event.timestamp.first_occurrence`**
:   Timestamp of first occurrence of event

type: date


**`kubernetes.event.timestamp.last_occurrence`**
:   Timestamp of last occurrence of event

type: date


**`kubernetes.event.message`**
:   Message recorded for the given event

type: text


**`kubernetes.event.reason`**
:   Reason recorded for the given event

type: keyword


**`kubernetes.event.type`**
:   Type of the given event

type: keyword



## source [_source_2]

The component reporting this event

**`kubernetes.event.source.component`**
:   Component from which the event is generated

type: keyword


**`kubernetes.event.source.host`**
:   Node name on which the event is generated

type: keyword



## metadata [_metadata_2]

Metadata associated with the given event

**`kubernetes.event.metadata.timestamp.created`**
:   Timestamp of creation of the given event

type: date


**`kubernetes.event.metadata.generate_name`**
:   Generate name of the event

type: keyword


**`kubernetes.event.metadata.name`**
:   Name of the event

type: keyword


**`kubernetes.event.metadata.namespace`**
:   Namespace in which event was generated

type: keyword


**`kubernetes.event.metadata.resource_version`**
:   Version of the event resource

type: keyword


**`kubernetes.event.metadata.uid`**
:   Unique identifier to the event object

type: keyword


**`kubernetes.event.metadata.self_link`**
:   URL representing the event

type: keyword



## involved_object [_involved_object]

Metadata associated with the given involved object

**`kubernetes.event.involved_object.api_version`**
:   API version of the object

type: keyword


**`kubernetes.event.involved_object.kind`**
:   API kind of the object

type: keyword


**`kubernetes.event.involved_object.name`**
:   name of the object

type: keyword


**`kubernetes.event.involved_object.resource_version`**
:   resource version of the object

type: keyword


**`kubernetes.event.involved_object.uid`**
:   UUID version of the object

type: keyword



## node [_node_4]

kubernetes node metrics

**`kubernetes.node.start_time`**
:   Start time

type: date



## cpu [_cpu_7]

CPU usage metrics

**`kubernetes.node.cpu.usage.core.ns`**
:   Node CPU Core usage nanoseconds

type: double


**`kubernetes.node.cpu.usage.nanocores`**
:   CPU used nanocores

type: double


**`kubernetes.node.memory.available.bytes`**
:   Total available memory

type: double

format: bytes


**`kubernetes.node.memory.usage.bytes`**
:   Total memory usage

type: double

format: bytes


**`kubernetes.node.memory.rss.bytes`**
:   RSS memory usage

type: double

format: bytes


**`kubernetes.node.memory.workingset.bytes`**
:   Working set memory usage

type: double

format: bytes


**`kubernetes.node.memory.pagefaults`**
:   Number of page faults

type: double


**`kubernetes.node.memory.majorpagefaults`**
:   Number of major page faults

type: double


**`kubernetes.node.network.rx.bytes`**
:   Received bytes on the default interface. If default interface is not defined, will be reported not correct value `0`

type: double

format: bytes


**`kubernetes.node.network.rx.errors`**
:   Rx errors on the default interface. If default interface is not defined, will be reported not correct value `0`

type: double


**`kubernetes.node.network.tx.bytes`**
:   Transmitted bytes on the default interface. If default interface is not defined, will be reported not correct value `0`

type: double

format: bytes


**`kubernetes.node.network.tx.errors`**
:   Tx errors on the default interface. If default interface is not defined, will be reported not correct value `0`

type: double


**`kubernetes.node.fs.capacity.bytes`**
:   Filesystem total capacity in bytes

type: double

format: bytes


**`kubernetes.node.fs.available.bytes`**
:   Filesystem total available in bytes

type: double

format: bytes


**`kubernetes.node.fs.used.bytes`**
:   Filesystem total used in bytes

type: double

format: bytes


**`kubernetes.node.fs.inodes.used`**
:   Number of used inodes

type: double


**`kubernetes.node.fs.inodes.count`**
:   Number of inodes

type: double


**`kubernetes.node.fs.inodes.free`**
:   Number of free inodes

type: double


**`kubernetes.node.runtime.imagefs.capacity.bytes`**
:   Image filesystem total capacity in bytes

type: double

format: bytes


**`kubernetes.node.runtime.imagefs.available.bytes`**
:   Image filesystem total available in bytes

type: double

format: bytes


**`kubernetes.node.runtime.imagefs.used.bytes`**
:   Image filesystem total used in bytes

type: double

format: bytes



## pod [_pod]

kubernetes pod metrics

**`kubernetes.pod.start_time`**
:   Start time

type: date


**`kubernetes.pod.network.rx.bytes`**
:   Received bytes

type: double

format: bytes


**`kubernetes.pod.network.rx.errors`**
:   Rx errors

type: double


**`kubernetes.pod.network.tx.bytes`**
:   Transmitted bytes

type: double

format: bytes


**`kubernetes.pod.network.tx.errors`**
:   Tx errors

type: double



## cpu [_cpu_8]

CPU usage metrics

**`kubernetes.pod.cpu.usage.nanocores`**
:   CPU used nanocores

type: double


**`kubernetes.pod.cpu.usage.node.pct`**
:   CPU usage as a percentage of the total node CPU

type: scaled_float

format: percent


**`kubernetes.pod.cpu.usage.limit.pct`**
:   CPU usage as a percentage of the defined cpu limits sum of the pod containers. If any container is missing a limit the metric is not emitted.

type: scaled_float

format: percent


**`kubernetes.pod.memory.usage.bytes`**
:   Total memory usage

type: double

format: bytes


**`kubernetes.pod.memory.usage.node.pct`**
:   Memory usage as a percentage of the total node allocatable memory

type: scaled_float

format: percent


**`kubernetes.pod.memory.usage.limit.pct`**
:   Memory usage as a percentage of the defined memory limits sum of the pod containers. If any container is missing a limit the metric is not emitted.

type: scaled_float

format: percent


**`kubernetes.pod.memory.available.bytes`**
:   Total memory available

type: double

format: bytes


**`kubernetes.pod.memory.working_set.bytes`**
:   Total working set memory

type: double

format: bytes


**`kubernetes.pod.memory.working_set.limit.pct`**
:   Working set memory usage as a percentage of the defined limits sum of the pod containers. If any container is missing a limit the metric is not emitted.

type: scaled_float

format: percent


**`kubernetes.pod.memory.rss.bytes`**
:   Total resident set size memory

type: double

format: bytes


**`kubernetes.pod.memory.page_faults`**
:   Total page faults

type: double


**`kubernetes.pod.memory.major_page_faults`**
:   Total major page faults

type: double



## proxy [_proxy_3]

Kubernetes proxy server metrics

**`kubernetes.proxy.code`**
:   HTTP code

type: keyword


**`kubernetes.proxy.method`**
:   HTTP method

type: keyword


**`kubernetes.proxy.host`**
:   HTTP host

type: keyword


**`kubernetes.proxy.verb`**
:   HTTP verb

type: keyword


**`kubernetes.proxy.process.cpu.sec`**
:   Total user and system CPU time spent in seconds

type: double


**`kubernetes.proxy.process.memory.resident.bytes`**
:   Bytes in resident memory

type: long

format: bytes


**`kubernetes.proxy.process.memory.virtual.bytes`**
:   Bytes in virtual memory

type: long

format: bytes


**`kubernetes.proxy.process.fds.open.count`**
:   Number of open file descriptors

type: long


**`kubernetes.proxy.process.fds.max.count`**
:   Limit for open file descriptors

type: long


**`kubernetes.proxy.process.started.sec`**
:   Start time of the process since unix epoch in seconds

type: double


**`kubernetes.proxy.client.request.count`**
:   Number of HTTP requests to API server, broken down by status code, method and host

type: long


**`kubernetes.proxy.client.request.duration.us.sum`**
:   Sum of requests latency in microseconds, broken down by verb and host

type: long


**`kubernetes.proxy.client.request.duration.us.count`**
:   Number of request duration operations to API server, broken down by verb and host

type: long


**`kubernetes.proxy.client.request.duration.us.bucket.*`**
:   Requests latency distribution in histogram buckets, broken down by verb and host

type: object


**`kubernetes.proxy.client.request.size.bytes.sum`**
:   Requests size sum in bytes, broken down by verb and host

type: long

format: bytes


**`kubernetes.proxy.client.request.size.bytes.count`**
:   Number of requests, broken down by verb and host

type: long


**`kubernetes.proxy.client.request.size.bytes.bucket.*`**
:   Requests size distribution in histogram buckets, broken down by verb and host

type: object


**`kubernetes.proxy.client.response.size.bytes.count`**
:   Number of responses, broken down by verb and host

type: long


**`kubernetes.proxy.client.response.size.bytes.sum`**
:   Responses size sum in bytes, broken down by verb and host

type: long

format: bytes


**`kubernetes.proxy.client.response.size.bytes.bucket.*`**
:   Responses size distribution in histogram buckets, broken down by verb and host

type: object



## sync [_sync]

kubeproxy proxy sync metrics

**`kubernetes.proxy.sync.rules.duration.us.sum`**
:   SyncProxyRules latency sum in microseconds

type: long


**`kubernetes.proxy.sync.rules.duration.us.count`**
:   Number of SyncProxyRules latency operations

type: long


**`kubernetes.proxy.sync.rules.duration.us.bucket.*`**
:   SyncProxyRules latency distribution in histogram buckets

type: object


**`kubernetes.proxy.sync.networkprogramming.duration.us.sum`**
:   Sum of network programming latency in microseconds

type: long


**`kubernetes.proxy.sync.networkprogramming.duration.us.count`**
:   Number of network programming latency operations

type: long


**`kubernetes.proxy.sync.networkprogramming.duration.us.bucket.*`**
:   Network programming latency distribution in histogram buckets

type: object



## scheduler [_scheduler]

Kubernetes scheduler metrics

**`kubernetes.scheduler.verb`**
:   HTTP verb

type: keyword


**`kubernetes.scheduler.host`**
:   HTTP host

type: keyword


**`kubernetes.scheduler.code`**
:   HTTP code

type: keyword


**`kubernetes.scheduler.method`**
:   HTTP method

type: keyword


**`kubernetes.scheduler.queue`**
:   Scheduling queue

type: keyword


**`kubernetes.scheduler.event`**
:   Scheduling event

type: keyword


**`kubernetes.scheduler.profile`**
:   Scheduling profile

type: keyword


**`kubernetes.scheduler.result`**
:   Attempt result to schedule pod

type: keyword


**`kubernetes.scheduler.name`**
:   Name for the resource

type: keyword


**`kubernetes.scheduler.leader.is_master`**
:   Whether the scheduler instance is leader

type: boolean


**`kubernetes.scheduler.process.cpu.sec`**
:   Total user and system CPU time spent in seconds

type: double


**`kubernetes.scheduler.process.memory.resident.bytes`**
:   Bytes in resident memory

type: long

format: bytes


**`kubernetes.scheduler.process.memory.virtual.bytes`**
:   Bytes in virtual memory

type: long

format: bytes


**`kubernetes.scheduler.process.fds.open.count`**
:   Number of open file descriptors

type: long


**`kubernetes.scheduler.process.fds.max.count`**
:   Limit for open file descriptors

type: long


**`kubernetes.scheduler.process.started.sec`**
:   Start time of the process since unix epoch in seconds

type: double


**`kubernetes.scheduler.client.request.count`**
:   Number of HTTP requests to API server, broken down by status code, method and host

type: long


**`kubernetes.scheduler.client.request.duration.us.sum`**
:   Sum of requests latency in microseconds, broken down by verb and host

type: long


**`kubernetes.scheduler.client.request.duration.us.count`**
:   Number of request duration operations to API server, broken down by verb and host

type: long


**`kubernetes.scheduler.client.request.duration.us.bucket.*`**
:   Requests latency distribution in histogram buckets, broken down by verb and host

type: object


**`kubernetes.scheduler.client.request.size.bytes.sum`**
:   Requests size sum in bytes, broken down by verb and host

type: long

format: bytes


**`kubernetes.scheduler.client.request.size.bytes.count`**
:   Number of requests, broken down by verb and host

type: long


**`kubernetes.scheduler.client.request.size.bytes.bucket.*`**
:   Requests size distribution in histogram buckets, broken down by verb and host

type: object


**`kubernetes.scheduler.client.response.size.bytes.count`**
:   Number of responses, broken down by verb and host

type: long


**`kubernetes.scheduler.client.response.size.bytes.sum`**
:   Responses size sum in bytes, broken down by verb and host

type: long

format: bytes


**`kubernetes.scheduler.client.response.size.bytes.bucket.*`**
:   Responses size distribution in histogram buckets, broken down by verb and host

type: object


**`kubernetes.scheduler.workqueue.longestrunning.sec`**
:   How many seconds has the longest running processor been running, broken down by workqueue name

type: double


**`kubernetes.scheduler.workqueue.unfinished.sec`**
:   How many seconds of work has done that is in progress and hasn’t been considered in the longest running processor, broken down by workqueue name

type: double


**`kubernetes.scheduler.workqueue.adds.count`**
:   Workqueue add count, broken down by workqueue name

type: long


**`kubernetes.scheduler.workqueue.depth.count`**
:   Workqueue current depth, broken down by workqueue name

type: long


**`kubernetes.scheduler.workqueue.retries.count`**
:   Workqueue number of retries, broken down by workqueue name

type: long


**`kubernetes.scheduler.scheduling.pending.pods.count`**
:   Number of current pending pods, broken down by the queue type

type: long


**`kubernetes.scheduler.scheduling.preemption.victims.bucket.*`**
:   Number of preemption victims distribution in histogram buckets

type: object


**`kubernetes.scheduler.scheduling.preemption.victims.sum`**
:   Preemption victims sum

type: long


**`kubernetes.scheduler.scheduling.preemption.victims.count`**
:   Number of preemption victims

type: long


**`kubernetes.scheduler.scheduling.preemption.attempts.count`**
:   Total preemption attempts in the cluster so far

type: long


**`kubernetes.scheduler.scheduling.attempts.duration.us.bucket.*`**
:   Scheduling attempt latency distribution in histogram buckets, broken down by profile and result

type: object


**`kubernetes.scheduler.scheduling.attempts.duration.us.sum`**
:   Sum of scheduling attempt latency in microseconds, broken down by profile and result

type: long


**`kubernetes.scheduler.scheduling.attempts.duration.us.count`**
:   Number of scheduling attempts, broken down by profile and result

type: long



## container [_container_6]

kubernetes container metrics

**`kubernetes.container.id`**
:   Container id

type: keyword


**`kubernetes.container.status.phase`**
:   Container phase (running, waiting, terminated)

type: keyword


**`kubernetes.container.status.ready`**
:   Container ready status

type: boolean


**`kubernetes.container.status.restarts`**
:   Container restarts count

type: integer


**`kubernetes.container.status.reason`**
:   The reason the container is currently in waiting (ContainerCreating, CrashLoopBackoff, ErrImagePull, ImagePullBackoff) or terminated (Completed, ContainerCannotRun, Error, OOMKilled) state.

type: keyword


**`kubernetes.container.status.last_terminated_reason`**
:   The last reason the container was in terminated state (Completed, ContainerCannotRun, Error or OOMKilled).

type: keyword


**`kubernetes.container.status.last_terminated_timestamp`**
:   Last terminated time (epoch) of the container

type: double


**`kubernetes.container.cpu.limit.cores`**
:   Container CPU cores limit

type: float


**`kubernetes.container.cpu.request.cores`**
:   Container CPU requested cores

type: float


**`kubernetes.container.memory.limit.bytes`**
:   Container memory limit in bytes

type: long

format: bytes


**`kubernetes.container.memory.request.bytes`**
:   Container requested memory in bytes

type: long

format: bytes



## cronjob [_cronjob]

kubernetes cronjob metrics

**`kubernetes.cronjob.name`**
:   Cronjob name

type: keyword


**`kubernetes.cronjob.schedule`**
:   Cronjob schedule

type: keyword


**`kubernetes.cronjob.concurrency`**
:   Concurrency policy

type: keyword


**`kubernetes.cronjob.active.count`**
:   Number of active pods for the cronjob

type: long


**`kubernetes.cronjob.is_suspended`**
:   Whether the cronjob is suspended

type: boolean


**`kubernetes.cronjob.created.sec`**
:   Epoch seconds since the cronjob was created

type: double


**`kubernetes.cronjob.last_schedule.sec`**
:   Epoch seconds for last cronjob run

type: double


**`kubernetes.cronjob.next_schedule.sec`**
:   Epoch seconds for next cronjob run

type: double


**`kubernetes.cronjob.deadline.sec`**
:   Deadline seconds after schedule for considering failed

type: long



## daemonset [_daemonset]

Kubernetes DaemonSet metrics

**`kubernetes.daemonset.name`**
:   type: keyword



## replicas [_replicas]

Kubernetes DaemonSet replica metrics

**`kubernetes.daemonset.replicas.available`**
:   The number of available replicas per DaemonSet

type: long


**`kubernetes.daemonset.replicas.desired`**
:   The desired number of replicas per DaemonSet

type: long


**`kubernetes.daemonset.replicas.ready`**
:   The number of ready replicas per DaemonSet

type: long


**`kubernetes.daemonset.replicas.unavailable`**
:   The number of unavailable replicas per DaemonSet

type: long



## deployment [_deployment_2]

kubernetes deployment metrics

**`kubernetes.deployment.paused`**
:   Kubernetes deployment paused status

type: boolean


**`kubernetes.deployment.status.available`**
:   Deployment Available Condition status (true, false or unknown)

type: keyword


**`kubernetes.deployment.status.progressing`**
:   Deployment Progresing Condition status (true, false or unknown)

type: keyword



## replicas [_replicas_2]

Kubernetes deployment replicas info

**`kubernetes.deployment.replicas.desired`**
:   Deployment number of desired replicas (spec)

type: integer


**`kubernetes.deployment.replicas.available`**
:   Deployment available replicas

type: integer


**`kubernetes.deployment.replicas.unavailable`**
:   Deployment unavailable replicas

type: integer


**`kubernetes.deployment.replicas.updated`**
:   Deployment updated replicas

type: integer



## job [_job]

Kubernetes job metrics

**`kubernetes.job.name`**
:   The name of the job resource

type: keyword



## pods [_pods]

Pod metrics for the job

**`kubernetes.job.pods.active`**
:   Number of active pods

type: long


**`kubernetes.job.pods.failed`**
:   Number of failed pods

type: long


**`kubernetes.job.pods.succeeded`**
:   Number of successful pods

type: long



## time [_time]

Kubernetes job timestamps

**`kubernetes.job.time.created`**
:   The time at which the job was created

type: date


**`kubernetes.job.time.completed`**
:   The time at which the job completed

type: date



## completions [_completions]

Kubernetes job completion settings

**`kubernetes.job.completions.desired`**
:   The configured completion count for the job (Spec)

type: long



## parallelism [_parallelism]

Kubernetes job parallelism settings

**`kubernetes.job.parallelism.desired`**
:   The configured parallelism of the job (Spec)

type: long



## owner [_owner]

Kubernetes job owner information

**`kubernetes.job.owner.name`**
:   The name of the resource that owns this job

type: keyword


**`kubernetes.job.owner.kind`**
:   The kind of resource that owns this job (eg. "CronJob")

type: keyword


**`kubernetes.job.owner.is_controller`**
:   Owner is controller ("true", "false", or "<none>")

type: keyword



## status [_status_3]

Kubernetes job status information

**`kubernetes.job.status.complete`**
:   Whether the job completed ("true", "false", or "unknown")

type: keyword


**`kubernetes.job.status.failed`**
:   Whether the job failed ("true", "false", or "unknown")

type: keyword



## state_namespace [_state_namespace]

Kubernetes namespace metrics.

**`kubernetes.state_namespace.created.sec`**
:   Unix creation timestamp.

type: double


**`kubernetes.state_namespace.status.active`**
:   Whether the namespace is active (true or false).

type: boolean


**`kubernetes.state_namespace.status.terminating`**
:   Whether the namespace is terminating (true or false).

type: boolean



## node [_node_5]

kubernetes node metrics

**`kubernetes.node.status.ready`**
:   Node ready status (true, false or unknown)

type: keyword


**`kubernetes.node.status.unschedulable`**
:   Node unschedulable status

type: boolean


**`kubernetes.node.status.memory_pressure`**
:   Node MemoryPressure status (true, false or unknown)

type: keyword


**`kubernetes.node.status.disk_pressure`**
:   Node DiskPressure status (true, false or unknown)

type: keyword


**`kubernetes.node.status.out_of_disk`**
:   Node OutOfDisk status (true, false or unknown)

type: keyword


**`kubernetes.node.status.pid_pressure`**
:   Node PIDPressure status (true, false or unknown)

type: keyword


**`kubernetes.node.status.network_unavailable`**
:   Node NetworkUnavailable status (true, false or unknown)

type: keyword


**`kubernetes.node.cpu.allocatable.cores`**
:   The allocatable CPU cores of a node that are available for pods scheduling

type: float


**`kubernetes.node.cpu.capacity.cores`**
:   Node CPU capacity cores

type: long


**`kubernetes.node.memory.allocatable.bytes`**
:   The allocatable memory of a node in bytes that is available for pods scheduling

type: long

format: bytes


**`kubernetes.node.memory.capacity.bytes`**
:   Node memory capacity in bytes

type: long

format: bytes


**`kubernetes.node.pod.allocatable.total`**
:   Node allocatable pods

type: long


**`kubernetes.node.pod.capacity.total`**
:   Node pod capacity

type: long


**`kubernetes.node.kubelet.version`**
:   Kubelet version.

type: keyword



## persistentvolume [_persistentvolume]

kubernetes persistent volume metrics from kube-state-metrics

**`kubernetes.persistentvolume.name`**
:   Volume name.

type: keyword


**`kubernetes.persistentvolume.capacity.bytes`**
:   Volume capacity

type: long


**`kubernetes.persistentvolume.phase`**
:   Volume phase according to kubernetes

type: keyword


**`kubernetes.persistentvolume.storage_class`**
:   Storage class for the volume

type: keyword



## persistentvolumeclaim [_persistentvolumeclaim]

kubernetes persistent volume claim metrics from kube-state-metrics

**`kubernetes.persistentvolumeclaim.name`**
:   PVC name.

type: keyword


**`kubernetes.persistentvolumeclaim.volume_name`**
:   Binded volume name.

type: keyword


**`kubernetes.persistentvolumeclaim.request_storage.bytes`**
:   Requested capacity.

type: long


**`kubernetes.persistentvolumeclaim.phase`**
:   PVC phase.

type: keyword


**`kubernetes.persistentvolumeclaim.access_mode`**
:   Access mode.

type: keyword


**`kubernetes.persistentvolumeclaim.storage_class`**
:   Storage class for the PVC.

type: keyword


**`kubernetes.persistentvolumeclaim.created`**
:   PersistentVolumeClaim creation date

type: date



## pod [_pod_2]

kubernetes pod metrics

**`kubernetes.pod.host_ip`**
:   Kubernetes pod host IP

type: ip



## status [_status_4]

Kubernetes pod status metrics

**`kubernetes.pod.status.phase`**
:   Kubernetes pod phase (Running, Pending…​)

type: keyword


**`kubernetes.pod.status.ready`**
:   Kubernetes pod ready status (true, false or unknown)

type: keyword


**`kubernetes.pod.status.scheduled`**
:   Kubernetes pod scheduled status (true, false, unknown)

type: keyword


**`kubernetes.pod.status.reason`**
:   The reason the pod is in its current state (Evicted, NodeAffinity, NodeLost, Shutdown or UnexpectedAdmissionError)

type: keyword


**`kubernetes.pod.status.ready_time`**
:   Readiness achieved time in unix timestamp for a pod

type: double



## replicaset [_replicaset]

kubernetes replica set metrics


## replicas [_replicas_3]

Kubernetes replica set paused status

**`kubernetes.replicaset.replicas.available`**
:   The number of replicas per ReplicaSet

type: long


**`kubernetes.replicaset.replicas.desired`**
:   The number of replicas per ReplicaSet

type: long


**`kubernetes.replicaset.replicas.ready`**
:   The number of ready replicas per ReplicaSet

type: long


**`kubernetes.replicaset.replicas.observed`**
:   The generation observed by the ReplicaSet controller

type: long


**`kubernetes.replicaset.replicas.labeled`**
:   The number of fully labeled replicas per ReplicaSet

type: long



## resourcequota [_resourcequota]

kubernetes resourcequota metrics

**`kubernetes.resourcequota.created.sec`**
:   Epoch seconds since the ResourceQuota was created

type: double


**`kubernetes.resourcequota.quota`**
:   Quota informed (hard or used) for the resource

type: double


**`kubernetes.resourcequota.name`**
:   ResourceQuota name

type: keyword


**`kubernetes.resourcequota.type`**
:   Quota information type, `hard` or `used`

type: keyword


**`kubernetes.resourcequota.resource`**
:   Resource name the quota applies to

type: keyword



## service [_service_3]

kubernetes service metrics

**`kubernetes.service.name`**
:   Service name.

type: keyword


**`kubernetes.service.cluster_ip`**
:   Internal IP for the service.

type: keyword


**`kubernetes.service.external_name`**
:   Service external DNS name

type: keyword


**`kubernetes.service.external_ip`**
:   Service external IP

type: keyword


**`kubernetes.service.load_balancer_ip`**
:   Load Balancer service IP

type: keyword


**`kubernetes.service.type`**
:   Service type

type: keyword


**`kubernetes.service.ingress_ip`**
:   Ingress IP

type: keyword


**`kubernetes.service.ingress_hostname`**
:   Ingress Hostname

type: keyword


**`kubernetes.service.created`**
:   Service creation date

type: date



## statefulset [_statefulset]

kubernetes stateful set metrics

**`kubernetes.statefulset.created`**
:   The creation timestamp (epoch) for StatefulSet

type: long



## replicas [_replicas_4]

Kubernetes stateful set replicas status

**`kubernetes.statefulset.replicas.observed`**
:   The number of observed replicas per StatefulSet

type: long


**`kubernetes.statefulset.replicas.desired`**
:   The number of desired replicas per StatefulSet

type: long


**`kubernetes.statefulset.replicas.ready`**
:   The number of ready replicas per StatefulSet

type: long



## generation [_generation]

Kubernetes stateful set generation information

**`kubernetes.statefulset.generation.observed`**
:   The observed generation per StatefulSet

type: long


**`kubernetes.statefulset.generation.desired`**
:   The desired generation per StatefulSet

type: long



## storageclass [_storageclass]

kubernetes storage class metrics

**`kubernetes.storageclass.name`**
:   Storage class name.

type: keyword


**`kubernetes.storageclass.provisioner`**
:   Volume provisioner for the storage class.

type: keyword


**`kubernetes.storageclass.reclaim_policy`**
:   Reclaim policy for dynamically created volumes

type: keyword


**`kubernetes.storageclass.volume_binding_mode`**
:   Mode for default provisioning and binding

type: keyword


**`kubernetes.storageclass.created`**
:   Storage class creation date

type: date



## system [_system_3]

kubernetes system containers metrics

**`kubernetes.system.container`**
:   Container name

type: keyword


**`kubernetes.system.start_time`**
:   Start time

type: date



## cpu [_cpu_9]

CPU usage metrics

**`kubernetes.system.cpu.usage.core.ns`**
:   CPU Core usage nanoseconds

type: double


**`kubernetes.system.cpu.usage.nanocores`**
:   CPU used nanocores

type: double


**`kubernetes.system.memory.usage.bytes`**
:   Total memory usage

type: double

format: bytes


**`kubernetes.system.memory.rss.bytes`**
:   RSS memory usage

type: double

format: bytes


**`kubernetes.system.memory.workingset.bytes`**
:   Working set memory usage

type: double

format: bytes


**`kubernetes.system.memory.pagefaults`**
:   Number of page faults

type: double


**`kubernetes.system.memory.majorpagefaults`**
:   Number of major page faults

type: double



## volume [_volume]

kubernetes volume metrics

**`kubernetes.volume.name`**
:   Volume name

type: keyword


**`kubernetes.volume.fs.capacity.bytes`**
:   Filesystem total capacity in bytes

type: double

format: bytes


**`kubernetes.volume.fs.available.bytes`**
:   Filesystem total available in bytes

type: double

format: bytes


**`kubernetes.volume.fs.used.bytes`**
:   Filesystem total used in bytes

type: double

format: bytes


**`kubernetes.volume.fs.used.pct`**
:   Percentage of used storage

type: scaled_float

format: percent


**`kubernetes.volume.fs.inodes.used`**
:   Used inodes

type: double


**`kubernetes.volume.fs.inodes.free`**
:   Free inodes

type: double


**`kubernetes.volume.fs.inodes.count`**
:   Total inodes

type: double


**`kubernetes.volume.fs.inodes.pct`**
:   Percentage of used inodes

type: scaled_float

format: percent


