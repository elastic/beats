# Kube-state-metrics/Cronjob

### Version history

**January 2023**: Kube state metrics versions 2.4.2-2.7.0

### Resources

- [State cronjob metrics](https://github.com/kubernetes/kube-state-metrics/blob/main/internal/store/cronjob.go):
  declaration and description

### Metrics insight

All metrics have the labels:
- namespace
- cronjob

Metrics:
- kube_cronjob_info
  - schedule
  - concurrency_policy
- kube_cronjob_created
- kube_cronjob_status_last_schedule_time
- kube_cronjob_next_schedule_time
- kube_cronjob_spec_suspend
- kube_cronjob_spec_starting_deadline_seconds


### Setup environment for manual tests
Go to `metricbeat/module/kubernetes/_meta/test/docs`.






