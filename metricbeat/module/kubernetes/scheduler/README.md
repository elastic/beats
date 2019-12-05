# Scheduler Stats

## Version history

- June 2019, `v1.14.0`

## Resources

https://github.com/kubernetes/kubernetes/blob/master/pkg/scheduler/metrics/metrics.go

## Metrics insight

- leader_election_master_status
    - name
- scheduler_binding_duration_seconds_bucket
- scheduler_e2e_scheduling_duration_seconds_bucket
- scheduler_pod_preemption_victims
- scheduler_schedule_attempts_total
  - result
- scheduler_scheduling_algorithm_duration_seconds_bucket
- scheduler_scheduling_algorithm_predicate_evaluation_seconds_bucket
- scheduler_scheduling_algorithm_preemption_evaluation_seconds_bucket
- scheduler_scheduling_algorithm_priority_evaluation_seconds_bucket
- scheduler_scheduling_duration_seconds
  - operation
- scheduler_volume_scheduling_duration_seconds_bucket
  - operation

## Setup environment for manual tests

Kubernetes scheduler will usually run at every master node, but that might not be the case. It could be executed as a host process or an in-cluster pod.

- If host process (for example, systemd), metricbeat should be running at that same node gathering data from the scheduler.
- If executing as a pod:
    - A metricbeat instance can be also executed using the same affinity and deployment object (deployment, daemonset, ...) as the kubernetes scheduler.
    - A metricbeat instance can be launched as a sidecar container












