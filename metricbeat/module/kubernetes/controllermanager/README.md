# Controller Manager Stats

## Version history

- December 2022, `v1.25.x`

## Resources

- [Process metrics](https://github.com/kubernetes/kubernetes/blob/master/vendor/github.com/prometheus/client_golang/prometheus/process_collector.go)
- [Rest client metrics](https://github.com/kubernetes/component-base/blob/master/metrics/prometheus/restclient/metrics.go)
- [Node collector metrics](https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/nodelifecycle/metrics.go)
- [Workqueue metrics](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/component-base/metrics/prometheus/workqueue/metrics.go)
- [Metrics general information](https://kubernetes.io/docs/reference/instrumentation/metrics/)


## Metrics insight

Metrics used are either stable (not explicit) or alpha (explicit).

- process_cpu_seconds_total
- process_resident_memory_bytes
- process_virtual_memory_bytes
- process_open_fds
- process_start_time_seconds
- process_max_fds


- rest_client_requests_total (alpha)
  - code
  - host
  - method
- rest_client_response_size_bytes (alpha)
  - host
  - verb
- rest_client_request_size_bytes (alpha)
  - host
  - verb
- rest_client_request_duration_seconds (alpha)
  - host
  - verb


- workqueue_longest_running_processor_seconds (alpha)
  - name
- workqueue_unfinished_work_seconds (alpha)
  - name
- workqueue_adds_total (alpha)
  - name
- workqueue_depth (alpha)
  - name
- workqueue_retries_total (alpha)
  - name
- workqueue_work_duration_seconds (alpha)
  - name


- node_collector_evictions_total
  - zone
- node_collector_unhealthy_nodes_in_zone (alpha)
  - zone
- node_collector_zone_size (alpha)
  - zone


- leader_election_master_status (alpha)
  - name

## Setup environment for manual tests

WIP: controller manager will usually run at every master node, but that might not be the case. It could be executed as a host process or an in-cluster pod.

- If host process (for example, systemd), metricbeat should be running at that same node gathering data from the controller.
- If executing as a pod:
    - A metricbeat instance can be also executed using the same affinity and deployment object (deployment, daemonset, ...) as the controller manager.












