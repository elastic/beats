---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-gcp.html
---

# Google Cloud Platform fields [exported-fields-gcp]

GCP module

**`gcp.labels`**
:   GCP monitoring metrics labels

type: object


**`gcp.metrics.*.*.*.*`**
:   Metrics that returned from Google Cloud API query.

type: object



## billing [_billing_6]

Google Cloud Billing metrics

**`gcp.billing.cost_type`**
:   Cost types include regular, tax, adjustment, and rounding_error.

type: keyword


**`gcp.billing.invoice_month`**
:   Billing report month.

type: keyword


**`gcp.billing.project_id`**
:   Project ID of the billing report belongs to.

type: keyword


**`gcp.billing.total`**
:   Total billing amount.

type: float


**`gcp.billing.sku_id`**
:   The ID of the resource used by the service.

type: keyword


**`gcp.billing.sku_description`**
:   A description of the resource type used by the service. For example, a resource type for Cloud Storage is Standard Storage US.

type: keyword


**`gcp.billing.service_id`**
:   The ID of the service that the usage is associated with.

type: keyword


**`gcp.billing.service_description`**
:   The Google Cloud service that reported the Cloud Billing data.

type: keyword


**`gcp.billing.tags`**
:   A collection of key-value pairs that provide additional metadata.

type: nested


**`gcp.billing.effective_price`**
:   The charged price for usage of the Google Cloud SKUs and SKU tiers. Reflects contract pricing if applicable, otherwise, it’s the list price.

type: float



## carbon [_carbon]

Google Cloud Carbon Footprint metrics

**`gcp.carbon.project_id`**
:   Project ID the carbon footprint report belongs to.

type: keyword


**`gcp.carbon.project_name`**
:   Project name the carbon footprint report belongs to.

type: keyword


**`gcp.carbon.service_id`**
:   Service ID for the carbon footprint usage.

type: keyword


**`gcp.carbon.service_description`**
:   Service description for the carbon footprint usage.

type: keyword


**`gcp.carbon.region`**
:   Region for the carbon fooprint usage.

type: keyword


**`gcp.carbon.footprint.scope1`**
:   Scope 1 carbon footprint.

type: float


**`gcp.carbon.footprint.scope2.location`**
:   Scope 2 carbon footprint using location-based methodology.

type: float


**`gcp.carbon.footprint.scope2.market`**
:   Scope 2 carbon footprint using market-based methodology.

type: float


**`gcp.carbon.footprint.scope3`**
:   Scope 3 carbon footprint.

type: float


**`gcp.carbon.footprint.offsets`**
:   Total carbon offsets.

type: float



## compute [_compute_2]

Google Cloud Compute metrics

**`gcp.compute.firewall.dropped.bytes`**
:   Incoming bytes dropped by the firewall

type: long


**`gcp.compute.firewall.dropped_packets_count.value`**
:   Incoming packets dropped by the firewall

type: long


**`gcp.compute.instance.cpu.reserved_cores.value`**
:   Number of cores reserved on the host of the instance

type: double


**`gcp.compute.instance.cpu.usage_time.sec`**
:   Usage for all cores in seconds

type: double


**`gcp.compute.instance.cpu.usage.pct`**
:   The fraction of the allocated CPU that is currently in use on the instance

type: double


**`gcp.compute.instance.disk.read.bytes`**
:   Count of bytes read from disk

type: long


**`gcp.compute.instance.disk.read_ops_count.value`**
:   Count of disk read IO operations

type: long


**`gcp.compute.instance.disk.write.bytes`**
:   Count of bytes written to disk

type: long


**`gcp.compute.instance.disk.write_ops_count.value`**
:   Count of disk write IO operations

type: long


**`gcp.compute.instance.memory.balloon.ram_size.value`**
:   The total amount of memory in the VM. This metric is only available for VMs that belong to the e2 family.

type: long


**`gcp.compute.instance.memory.balloon.ram_used.value`**
:   Memory currently used in the VM. This metric is only available for VMs that belong to the e2 family.

type: long


**`gcp.compute.instance.memory.balloon.swap_in.bytes`**
:   The amount of memory read into the guest from its own swap space. This metric is only available for VMs that belong to the e2 family.

type: long


**`gcp.compute.instance.memory.balloon.swap_out.bytes`**
:   The amount of memory written from the guest to its own swap space. This metric is only available for VMs that belong to the e2 family.

type: long


**`gcp.compute.instance.network.ingress.bytes`**
:   Count of bytes received from the network

type: long


**`gcp.compute.instance.network.ingress.packets.count`**
:   Count of packets received from the network

type: long


**`gcp.compute.instance.network.egress.bytes`**
:   Count of bytes sent over the network

type: long


**`gcp.compute.instance.network.egress.packets.count`**
:   Count of packets sent over the network

type: long


**`gcp.compute.instance.uptime.sec`**
:   Number of seconds the VM has been running.

type: long


**`gcp.compute.instance.uptime_total.sec`**
:   Elapsed time since the VM was started, in seconds. Sampled every 60 seconds. After sampling, data is not visible for up to 120 seconds.

type: long



## dataproc [_dataproc]

Google Cloud Dataproc metrics

**`gcp.dataproc.cluster.hdfs.datanodes.count`**
:   Indicates the number of HDFS DataNodes that are running inside a cluster.

type: long


**`gcp.dataproc.cluster.hdfs.storage_capacity.value`**
:   Indicates capacity of HDFS system running on cluster in GB.

type: double


**`gcp.dataproc.cluster.hdfs.storage_utilization.value`**
:   The percentage of HDFS storage currently used.

type: double


**`gcp.dataproc.cluster.hdfs.unhealthy_blocks.count`**
:   Indicates the number of unhealthy blocks inside the cluster.

type: long


**`gcp.dataproc.cluster.job.failed.count`**
:   Indicates the number of jobs that have failed on a cluster.

type: long


**`gcp.dataproc.cluster.job.running.count`**
:   Indicates the number of jobs that are running on a cluster.

type: long


**`gcp.dataproc.cluster.job.submitted.count`**
:   Indicates the number of jobs that have been submitted to a cluster.

type: long


**`gcp.dataproc.cluster.operation.failed.count`**
:   Indicates the number of operations that have failed on a cluster.

type: long


**`gcp.dataproc.cluster.operation.running.count`**
:   Indicates the number of operations that are running on a cluster.

type: long


**`gcp.dataproc.cluster.operation.submitted.count`**
:   Indicates the number of operations that have been submitted to a cluster.

type: long


**`gcp.dataproc.cluster.yarn.allocated_memory_percentage.value`**
:   The percentage of YARN memory is allocated.

type: double


**`gcp.dataproc.cluster.yarn.apps.count`**
:   Indicates the number of active YARN applications.

type: long


**`gcp.dataproc.cluster.yarn.containers.count`**
:   Indicates the number of YARN containers.

type: long


**`gcp.dataproc.cluster.yarn.memory_size.value`**
:   Indicates the YARN memory size in GB.

type: double


**`gcp.dataproc.cluster.yarn.nodemanagers.count`**
:   Indicates the number of YARN NodeManagers running inside cluster.

type: long


**`gcp.dataproc.cluster.yarn.pending_memory_size.value`**
:   The current memory request, in GB, that is pending to be fulfilled by the scheduler.

type: double


**`gcp.dataproc.cluster.yarn.virtual_cores.count`**
:   Indicates the number of virtual cores in YARN.

type: long


**`gcp.dataproc.cluster.job.completion_time.value`**
:   The time jobs took to complete from the time the user submits a job to the time Dataproc reports it is completed.

type: object


**`gcp.dataproc.cluster.job.duration.value`**
:   The time jobs have spent in a given state.

type: object


**`gcp.dataproc.cluster.operation.completion_time.value`**
:   The time operations took to complete from the time the user submits a operation to the time Dataproc reports it is completed.

type: object


**`gcp.dataproc.cluster.operation.duration.value`**
:   The time operations have spent in a given state.

type: object



## firestore [_firestore]

Google Cloud Firestore metrics

**`gcp.firestore.document.delete.count`**
:   The number of successful document deletes.

type: long


**`gcp.firestore.document.read.count`**
:   The number of successful document reads from queries or lookups.

type: long


**`gcp.firestore.document.write.count`**
:   The number of successful document writes.

type: long



## gke [_gke_2]

`gke` contains the metrics that we scraped from GCP Stackdriver API containing monitoring metrics for GCP GKE

**`gcp.gke.container.cpu.core_usage_time.sec`**
:   Cumulative CPU usage on all cores used by the container in seconds. Sampled every 60 seconds.

type: double


**`gcp.gke.container.cpu.limit_cores.value`**
:   CPU cores limit of the container. Sampled every 60 seconds.

type: double


**`gcp.gke.container.cpu.limit_utilization.pct`**
:   The fraction of the CPU limit that is currently in use on the instance. This value cannot exceed 1 as usage cannot exceed the limit. Sampled every 60 seconds. After sampling, data is not visible for up to 240 seconds.

type: double


**`gcp.gke.container.cpu.request_cores.value`**
:   Number of CPU cores requested by the container. Sampled every 60 seconds. After sampling, data is not visible for up to 120 seconds.

type: double


**`gcp.gke.container.cpu.request_utilization.pct`**
:   The fraction of the requested CPU that is currently in use on the instance. This value can be greater than 1 as usage can exceed the request. Sampled every 60 seconds. After sampling, data is not visible for up to 240 seconds.

type: double


**`gcp.gke.container.ephemeral_storage.limit.bytes`**
:   Local ephemeral storage limit in bytes. Sampled every 60 seconds.

type: long


**`gcp.gke.container.ephemeral_storage.request.bytes`**
:   Local ephemeral storage request in bytes. Sampled every 60 seconds.

type: long


**`gcp.gke.container.ephemeral_storage.used.bytes`**
:   Local ephemeral storage usage in bytes. Sampled every 60 seconds.

type: long


**`gcp.gke.container.memory.limit.bytes`**
:   Memory limit of the container in bytes. Sampled every 60 seconds.

type: long


**`gcp.gke.container.memory.limit_utilization.pct`**
:   The fraction of the memory limit that is currently in use on the instance. This value cannot exceed 1 as usage cannot exceed the limit. Sampled every 60 seconds. After sampling, data is not visible for up to 120 seconds.

type: double


**`gcp.gke.container.memory.page_fault.count`**
:   Number of page faults, broken down by type, major and minor.

type: long


**`gcp.gke.container.memory.request.bytes`**
:   Memory request of the container in bytes. Sampled every 60 seconds. After sampling, data is not visible for up to 120 seconds.

type: long


**`gcp.gke.container.memory.request_utilization.pct`**
:   The fraction of the requested memory that is currently in use on the instance. This value can be greater than 1 as usage can exceed the request. Sampled every 60 seconds. After sampling, data is not visible for up to 240 seconds.

type: double


**`gcp.gke.container.memory.used.bytes`**
:   Memory usage in bytes. Sampled every 60 seconds.

type: long


**`gcp.gke.container.restart.count`**
:   Number of times the container has restarted. Sampled every 60 seconds. After sampling, data is not visible for up to 120 seconds.

type: long


**`gcp.gke.container.uptime.sec`**
:   Time in seconds that the container has been running. Sampled every 60 seconds.

type: double


**`gcp.gke.node.cpu.allocatable_cores.value`**
:   Number of allocatable CPU cores on the node. Sampled every 60 seconds.

type: double


**`gcp.gke.node.cpu.allocatable_utilization.pct`**
:   The fraction of the allocatable CPU that is currently in use on the instance. Sampled every 60 seconds. After sampling, data is not visible for up to 240 seconds.

type: double


**`gcp.gke.node.cpu.core_usage_time.sec`**
:   Cumulative CPU usage on all cores used on the node in seconds. Sampled every 60 seconds.

type: double


**`gcp.gke.node.cpu.total_cores.value`**
:   Total number of CPU cores on the node. Sampled every 60 seconds.

type: double


**`gcp.gke.node.ephemeral_storage.allocatable.bytes`**
:   Local ephemeral storage bytes allocatable on the node. Sampled every 60 seconds.

type: long


**`gcp.gke.node.ephemeral_storage.inodes_free.value`**
:   Free number of inodes on local ephemeral storage. Sampled every 60 seconds.

type: long


**`gcp.gke.node.ephemeral_storage.inodes_total.value`**
:   Total number of inodes on local ephemeral storage. Sampled every 60 seconds.

type: long


**`gcp.gke.node.ephemeral_storage.total.bytes`**
:   Total ephemeral storage bytes on the node. Sampled every 60 seconds.

type: long


**`gcp.gke.node.ephemeral_storage.used.bytes`**
:   Local ephemeral storage bytes used by the node. Sampled every 60 seconds.

type: long


**`gcp.gke.node.memory.allocatable.bytes`**
:   Cumulative memory bytes used by the node. Sampled every 60 seconds.

type: long


**`gcp.gke.node.memory.allocatable_utilization.pct`**
:   The fraction of the allocatable memory that is currently in use on the instance. This value cannot exceed 1 as usage cannot exceed allocatable memory bytes. Sampled every 60 seconds. After sampling, data is not visible for up to 120 seconds.

type: double


**`gcp.gke.node.memory.total.bytes`**
:   Number of bytes of memory allocatable on the node. Sampled every 60 seconds.

type: long


**`gcp.gke.node.memory.used.bytes`**
:   Cumulative memory bytes used by the node. Sampled every 60 seconds.

type: long


**`gcp.gke.node.network.received_bytes.count`**
:   Cumulative number of bytes received by the node over the network. Sampled every 60 seconds.

type: long


**`gcp.gke.node.network.sent_bytes.count`**
:   Cumulative number of bytes transmitted by the node over the network. Sampled every 60 seconds.

type: long


**`gcp.gke.node.pid_limit.value`**
:   The max PID of OS on the node. Sampled every 60 seconds.

type: long


**`gcp.gke.node.pid_used.value`**
:   The number of running process in the OS on the node. Sampled every 60 seconds.

type: long


**`gcp.gke.node_daemon.cpu.core_usage_time.sec`**
:   Cumulative CPU usage on all cores used by the node level system daemon in seconds. Sampled every 60 seconds.

type: double


**`gcp.gke.node_daemon.memory.used.bytes`**
:   Memory usage by the system daemon in bytes. Sampled every 60 seconds.

type: long


**`gcp.gke.pod.network.received.bytes`**
:   Cumulative number of bytes received by the pod over the network. Sampled every 60 seconds.

type: long


**`gcp.gke.pod.network.sent.bytes`**
:   Cumulative number of bytes transmitted by the pod over the network. Sampled every 60 seconds.

type: long


**`gcp.gke.pod.volume.total.bytes`**
:   Total number of disk bytes available to the pod. Sampled every 60 seconds. After sampling, data is not visible for up to 120 seconds.

type: long


**`gcp.gke.pod.volume.used.bytes`**
:   Number of disk bytes used by the pod. Sampled every 60 seconds.

type: long


**`gcp.gke.pod.volume.utilization.pct`**
:   The fraction of the volume that is currently being used by the instance. This value cannot be greater than 1 as usage cannot exceed the total available volume space. Sampled every 60 seconds. After sampling, data is not visible for up to 120 seconds.

type: double



## loadbalancing [_loadbalancing_2]

Google Cloud Load Balancing metrics

**`gcp.loadbalancing.https.backend_request.bytes`**
:   The number of bytes sent as requests from HTTP/S load balancer to backends.

type: long


**`gcp.loadbalancing.https.backend_request.count`**
:   The number of requests served by backends of HTTP/S load balancer.

type: long


**`gcp.loadbalancing.https.backend_response.bytes`**
:   The number of bytes sent as responses from backends (or cache) to external HTTP(S) load balancer.

type: long


**`gcp.loadbalancing.https.request.bytes`**
:   The number of bytes sent as requests from clients to HTTP/S load balancer.

type: long


**`gcp.loadbalancing.https.request.count`**
:   The number of requests served by HTTP/S load balancer.

type: long


**`gcp.loadbalancing.https.response.bytes`**
:   The number of bytes sent as responses from HTTP/S load balancer to clients.

type: long


**`gcp.loadbalancing.l3.external.egress.bytes`**
:   The number of bytes sent from external TCP/UDP network load balancer backend to client of the flow. For TCP flows it’s counting bytes on application stream only.

type: long


**`gcp.loadbalancing.l3.external.egress_packets.count`**
:   The number of packets sent from external TCP/UDP network load balancer backend to client of the flow.

type: long


**`gcp.loadbalancing.l3.external.ingress.bytes`**
:   The number of bytes sent from client to external TCP/UDP network load balancer backend. For TCP flows it’s counting bytes on application stream only.

type: long


**`gcp.loadbalancing.l3.external.ingress_packets.count`**
:   The number of packets sent from client to external TCP/UDP network load balancer backend.

type: long


**`gcp.loadbalancing.l3.internal.egress.bytes`**
:   The number of bytes sent from ILB backend to client (for TCP flows it’s counting bytes on application stream only).

type: long


**`gcp.loadbalancing.l3.internal.egress_packets.count`**
:   The number of packets sent from ILB backend to client of the flow.

type: long


**`gcp.loadbalancing.l3.internal.ingress.bytes`**
:   The number of bytes sent from client to ILB backend (for TCP flows it’s counting bytes on application stream only).

type: long


**`gcp.loadbalancing.l3.internal.ingress_packets.count`**
:   The number of packets sent from client to ILB backend.

type: long


**`gcp.loadbalancing.tcp_ssl_proxy.closed_connections.value`**
:   Number of connections that were terminated over TCP/SSL proxy.

type: long


**`gcp.loadbalancing.tcp_ssl_proxy.egress.bytes`**
:   Number of bytes sent from VM to client using proxy.

type: long


**`gcp.loadbalancing.tcp_ssl_proxy.ingress.bytes`**
:   Number of bytes sent from client to VM using proxy.

type: long


**`gcp.loadbalancing.tcp_ssl_proxy.new_connections.value`**
:   Number of connections that were created over TCP/SSL proxy.

type: long


**`gcp.loadbalancing.tcp_ssl_proxy.open_connections.value`**
:   Current number of outstanding connections through the TCP/SSL proxy.

type: long


**`gcp.loadbalancing.https.backend_latencies.value`**
:   A distribution of the latency calculated from when the request was sent by the proxy to the backend until the proxy received from the backend the last byte of response.

type: object


**`gcp.loadbalancing.https.external.regional.backend_latencies.value`**
:   A distribution of the latency calculated from when the request was sent by the proxy to the backend until the proxy received from the backend the last byte of response.

type: object


**`gcp.loadbalancing.https.external.regional.total_latencies.value`**
:   A distribution of the latency calculated from when the request was received by the proxy until the proxy got ACK from client on last response byte.

type: object


**`gcp.loadbalancing.https.frontend_tcp_rtt.value`**
:   A distribution of the RTT measured for each connection between client and proxy.

type: object


**`gcp.loadbalancing.https.internal.backend_latencies.value`**
:   A distribution of the latency calculated from when the request was sent by the internal HTTP/S load balancer proxy to the backend until the proxy received from the backend the last byte of response.

type: object


**`gcp.loadbalancing.https.internal.total_latencies.value`**
:   A distribution of the latency calculated from when the request was received by the internal HTTP/S load balancer proxy until the proxy got ACK from client on last response byte.

type: object


**`gcp.loadbalancing.https.total_latencies.value`**
:   A distribution of the latency calculated from when the request was received by the external HTTP/S load balancer proxy until the proxy got ACK from client on last response byte.

type: object


**`gcp.loadbalancing.l3.external.rtt_latencies.value`**
:   A distribution of the round trip time latency, measured over TCP connections for the external network load balancer.

type: object


**`gcp.loadbalancing.l3.internal.rtt_latencies.value`**
:   A distribution of RTT measured over TCP connections for internal TCP/UDP load balancer flows.

type: object


**`gcp.loadbalancing.tcp_ssl_proxy.frontend_tcp_rtt.value`**
:   A distribution of the smoothed RTT (in ms) measured by the proxy’s TCP stack, each minute application layer bytes pass from proxy to client.

type: object



## pubsub [_pubsub_2]

Google Cloud PubSub metrics

**`gcp.pubsub.snapshot.backlog.bytes`**
:   Total byte size of the messages retained in a snapshot.

type: long


**`gcp.pubsub.snapshot.backlog_bytes_by_region.bytes`**
:   Total byte size of the messages retained in a snapshot, broken down by Cloud region.

type: long


**`gcp.pubsub.snapshot.config_updates.count`**
:   Cumulative count of configuration changes, grouped by operation type and result.

type: long


**`gcp.pubsub.snapshot.num_messages.value`**
:   Number of messages retained in a snapshot.

type: long


**`gcp.pubsub.snapshot.num_messages_by_region.value`**
:   Number of messages retained in a snapshot, broken down by Cloud region.

type: long


**`gcp.pubsub.snapshot.oldest_message_age.sec`**
:   Age (in seconds) of the oldest message retained in a snapshot.

type: long


**`gcp.pubsub.snapshot.oldest_message_age_by_region.sec`**
:   Age (in seconds) of the oldest message retained in a snapshot, broken down by Cloud region.

type: long


**`gcp.pubsub.subscription.ack_message.count`**
:   Cumulative count of messages acknowledged by Acknowledge requests, grouped by delivery type.

type: long


**`gcp.pubsub.subscription.backlog.bytes`**
:   Total byte size of the unacknowledged messages (a.k.a. backlog messages) in a subscription.

type: long


**`gcp.pubsub.subscription.byte_cost.bytes`**
:   Cumulative cost of operations, measured in bytes. This is used to measure quota utilization.

type: long


**`gcp.pubsub.subscription.config_updates.count`**
:   Cumulative count of configuration changes for each subscription, grouped by operation type and result.

type: long


**`gcp.pubsub.subscription.dead_letter_message.count`**
:   Cumulative count of messages published to dead letter topic, grouped by result.

type: long


**`gcp.pubsub.subscription.mod_ack_deadline_message.count`**
:   Cumulative count of messages whose deadline was updated by ModifyAckDeadline requests, grouped by delivery type.

type: long


**`gcp.pubsub.subscription.mod_ack_deadline_message_operation.count`**
:   Cumulative count of ModifyAckDeadline message operations, grouped by result.

type: long


**`gcp.pubsub.subscription.mod_ack_deadline_request.count`**
:   Cumulative count of ModifyAckDeadline requests, grouped by result.

type: long


**`gcp.pubsub.subscription.num_outstanding_messages.value`**
:   Number of messages delivered to a subscription’s push endpoint, but not yet acknowledged.

type: long


**`gcp.pubsub.subscription.num_undelivered_messages.value`**
:   Number of unacknowledged messages (a.k.a. backlog messages) in a subscription.

type: long


**`gcp.pubsub.subscription.oldest_retained_acked_message_age.sec`**
:   Age (in seconds) of the oldest acknowledged message retained in a subscription.

type: long


**`gcp.pubsub.subscription.oldest_retained_acked_message_age_by_region.value`**
:   Age (in seconds) of the oldest acknowledged message retained in a subscription, broken down by Cloud region.

type: long


**`gcp.pubsub.subscription.oldest_unacked_message_age.sec`**
:   Age (in seconds) of the oldest unacknowledged message (a.k.a. backlog message) in a subscription.

type: long


**`gcp.pubsub.subscription.oldest_unacked_message_age_by_region.value`**
:   Age (in seconds) of the oldest unacknowledged message in a subscription, broken down by Cloud region.

type: long


**`gcp.pubsub.subscription.pull_ack_message_operation.count`**
:   Cumulative count of acknowledge message operations, grouped by result. For a definition of message operations, see Cloud Pub/Sub metric subscription/mod_ack_deadline_message_operation_count.

type: long


**`gcp.pubsub.subscription.pull_ack_request.count`**
:   Cumulative count of acknowledge requests, grouped by result.

type: long


**`gcp.pubsub.subscription.pull_message_operation.count`**
:   Cumulative count of pull message operations, grouped by result. For a definition of message operations, see Cloud Pub/Sub metric subscription/mod_ack_deadline_message_operation_count.

type: long


**`gcp.pubsub.subscription.pull_request.count`**
:   Cumulative count of pull requests, grouped by result.

type: long


**`gcp.pubsub.subscription.push_request.count`**
:   Cumulative count of push attempts, grouped by result. Unlike pulls, the push server implementation does not batch user messages. So each request only contains one user message. The push server retries on errors, so a given user message can appear multiple times.

type: long


**`gcp.pubsub.subscription.retained_acked.bytes`**
:   Total byte size of the acknowledged messages retained in a subscription.

type: long


**`gcp.pubsub.subscription.retained_acked_bytes_by_region.bytes`**
:   Total byte size of the acknowledged messages retained in a subscription, broken down by Cloud region.

type: long


**`gcp.pubsub.subscription.seek_request.count`**
:   Cumulative count of seek attempts, grouped by result.

type: long


**`gcp.pubsub.subscription.sent_message.count`**
:   Cumulative count of messages sent by Cloud Pub/Sub to subscriber clients, grouped by delivery type.

type: long


**`gcp.pubsub.subscription.streaming_pull_ack_message_operation.count`**
:   Cumulative count of StreamingPull acknowledge message operations, grouped by result. For a definition of message operations, see Cloud Pub/Sub metric subscription/mod_ack_deadline_message_operation_count.

type: long


**`gcp.pubsub.subscription.streaming_pull_ack_request.count`**
:   Cumulative count of streaming pull requests with non-empty acknowledge ids, grouped by result.

type: long


**`gcp.pubsub.subscription.streaming_pull_message_operation.count`**
:   Cumulative count of streaming pull message operations, grouped by result. For a definition of message operations, see Cloud Pub/Sub metric <code>subscription/mod_ack_deadline_message_operation_count

type: long


**`gcp.pubsub.subscription.streaming_pull_mod_ack_deadline_message_operation.count`**
:   Cumulative count of StreamingPull ModifyAckDeadline operations, grouped by result.

type: long


**`gcp.pubsub.subscription.streaming_pull_mod_ack_deadline_request.count`**
:   Cumulative count of streaming pull requests with non-empty ModifyAckDeadline fields, grouped by result.

type: long


**`gcp.pubsub.subscription.streaming_pull_response.count`**
:   Cumulative count of streaming pull responses, grouped by result.

type: long


**`gcp.pubsub.subscription.unacked_bytes_by_region.bytes`**
:   Total byte size of the unacknowledged messages in a subscription, broken down by Cloud region.

type: long


**`gcp.pubsub.topic.byte_cost.bytes`**
:   Cost of operations, measured in bytes. This is used to measure utilization for quotas.

type: long


**`gcp.pubsub.topic.config_updates.count`**
:   Cumulative count of configuration changes, grouped by operation type and result.

type: long


**`gcp.pubsub.topic.message_sizes.bytes`**
:   Distribution of publish message sizes (in bytes)

type: object


**`gcp.pubsub.topic.oldest_retained_acked_message_age_by_region.value`**
:   Age (in seconds) of the oldest acknowledged message retained in a topic, broken down by Cloud region.

type: long


**`gcp.pubsub.topic.oldest_unacked_message_age_by_region.value`**
:   Age (in seconds) of the oldest unacknowledged message in a topic, broken down by Cloud region.

type: long


**`gcp.pubsub.topic.retained_acked_bytes_by_region.bytes`**
:   Total byte size of the acknowledged messages retained in a topic, broken down by Cloud region.

type: long


**`gcp.pubsub.topic.send_message_operation.count`**
:   Cumulative count of publish message operations, grouped by result. For a definition of message operations, see Cloud Pub/Sub metric subscription/mod_ack_deadline_message_operation_count.

type: long


**`gcp.pubsub.topic.send_request.count`**
:   Cumulative count of publish requests, grouped by result.

type: long


**`gcp.pubsub.topic.streaming_pull_response.count`**
:   Cumulative count of streaming pull responses, grouped by result.

type: long


**`gcp.pubsub.topic.unacked_bytes_by_region.bytes`**
:   Total byte size of the unacknowledged messages in a topic, broken down by Cloud region.

type: long


**`gcp.pubsub.subscription.ack_latencies.value`**
:   Distribution of ack latencies in milliseconds. The ack latency is the time between when Cloud Pub/Sub sends a message to a subscriber client and when Cloud Pub/Sub receives an Acknowledge request for that message.

type: object


**`gcp.pubsub.subscription.push_request_latencies.value`**
:   Distribution of push request latencies (in microseconds), grouped by result.

type: object



## storage [_storage_3]

Google Cloud Storage metrics

**`gcp.storage.api.request.count`**
:   Delta count of API calls, grouped by the API method name and response code.

type: long


**`gcp.storage.authz.acl_based_object_access.count`**
:   Delta count of requests that result in an object being granted access solely due to object ACLs.

type: long


**`gcp.storage.authz.acl_operations.count`**
:   Usage of ACL operations broken down by type.

type: long


**`gcp.storage.authz.object_specific_acl_mutation.count`**
:   Delta count of changes made to object specific ACLs.

type: long


**`gcp.storage.network.received.bytes`**
:   Delta count of bytes received over the network, grouped by the API method name and response code.

type: long


**`gcp.storage.network.sent.bytes`**
:   Delta count of bytes sent over the network, grouped by the API method name and response code.

type: long


**`gcp.storage.storage.object.count`**
:   Total number of objects per bucket, grouped by storage class. This value is measured once per day, and the value is repeated at each sampling interval throughout the day.

type: long


**`gcp.storage.storage.total_byte_seconds.bytes`**
:   Delta count of bytes received over the network, grouped by the API method name and response code.

type: long


**`gcp.storage.storage.total.bytes`**
:   Total size of all objects in the bucket, grouped by storage class. This value is measured once per day, and the value is repeated at each sampling interval throughout the day.

type: long


