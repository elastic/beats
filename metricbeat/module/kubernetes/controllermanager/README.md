# Controller Manager Stats

## Version history

- June 2019, `v1.14.3`

## Resources

Each controller emits a set of metrics, there is no source file to reference but a set of source files that are gathered into a single metrics resource.

## Metrics insight

http_request_duration_microseconds The HTTP request latencies in microseconds. Summary
    - handler

http_requests_total Total number of HTTP requests made. Counter
    - code
    - handler
    - method

http_request_size_bytes The HTTP request sizes in bytes. Summary
    - handler

http_response_size_bytes The HTTP response sizes in bytes. Summary
    - handler

rest_client_requests_total Number of HTTP requests, partitioned by status code, method, and host. Counter
    - code
    - host
    - method

workqueue_longest_running_processor_seconds. Gauge
    - name: 

workqueue_unfinished_work_seconds: How many seconds of work has done that is in progress and hasn't been observed by work_duration. Large values indicate stuck threads. One can deduce the number of stuck threads by observing the rate at which this increases. Gauge
    - name: 

process_cpu_seconds_total: Total user and system CPU time spent in seconds.
process_open_fds Number of open file descriptors.
process_resident_memory_bytes Resident memory size in bytes.
process_start_time_seconds Start time of the process since unix epoch in seconds.
process_virtual_memory_bytes Virtual memory size in bytes

node_collector_evictions_number Number of Node evictions that happened since current instance of NodeController started.
    - zone

node_collector_unhealthy_nodes_in_zone Gauge measuring number of not Ready Nodes per zones.
    - zone

node_collector_zone_health measuring percentage of healthy nodes per zone.
    - zone

node_collector_zone_size: measuring number of registered Nodes per zones.
    - zone

leader_election_master_status


## Setup environment for manual tests

WIP: controller manager will usually run at every master node, but that might not be the case. It could be executed as a host process or an in-cluster pod.

- If host process (for example, systemd), metricbeat should be running at that same node gathering data from the controller.
- If executing as a pod:
    - A metricbeat instance can be also executed using the same affinity and deployment object (deployment, daemonset, ...) as the controller manager.












