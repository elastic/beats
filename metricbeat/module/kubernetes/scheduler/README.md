# Scheduler Stats

## Version history

- June 2019, `v1.14.0`

## Resources

https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/metrics/metrics.go

## Metrics insight

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


- leader_election_master_status (alpha)
  - name


- scheduler_pending_pods (stable)
  - queue
- scheduler_preemption_victims (stable)
- scheduler_preemption_attempts_total (stable)
- scheduler_scheduling_attempt_duration_seconds (stable)
  - profile
  - result
- scheduler_schedule_attempts_total (stable)
  - profile
  - result

## Setup environment for manual tests

Kubernetes scheduler will usually run at every master node, but that might not be the case. It could be executed as a host process or an in-cluster pod.

- If host process (for example, systemd), metricbeat should be running at that same node gathering data from the scheduler.
- If executing as a pod:
    - A metricbeat instance can be also executed using the same affinity and deployment object (deployment, daemonset, ...) as the kubernetes scheduler.
    - A metricbeat instance can be launched as a sidecar container












