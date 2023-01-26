### Version history

**January 2023**: Kube state metrics versions 2.4.2-2.7.0

### Resources

- [State job metrics](https://github.com/kubernetes/kube-state-metrics/blob/main/internal/store/job.go):
declaration and description

### Metrics insight

All metrics have the labels:
- namespace
- job_name

Additionally:
- kube_job_owner
  - owner_kind
  - owner_name
  - owner_is_controller
- kube_job_status_active
- kube_job_status_failed
- kube_job_status_succeededt
- kube_job_spec_completions
- kube_job_spec_parallelism
- kube_job_created
- kube_job_status_completion_time
- kube_job_complete
  - condition
- kube_job_failed
  - condition


### Setup environment for manual tests
Go to `metricbeat/module/kubernetes/_meta/test/docs`.
