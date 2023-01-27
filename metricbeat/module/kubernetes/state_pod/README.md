### Version history

**January 2023**: Kube state metrics versions 2.4.2-2.7.0

### Resources

- [State pod metrics](https://github.com/kubernetes/kube-state-metrics/blob/main/internal/store/pod.go):
declaration and description

### Metrics insight

All metrics have the labels:
- namespace
- pod
- uid

Additionally:
- kube_pod_info
  - host_ip
  - pod_ip
  - node
  - created_by_kind
  - created_by_name
  - priority_class
  - host_network
- kube_pod_status_phase
  - phase
- kube_pod_status_ready
  - condition
- kube_pod_status_scheduled
  - condition


### Setup environment for manual tests
Go to `metricbeat/module/kubernetes/_meta/test/docs`.

