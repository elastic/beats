## Version history

**January 2023**: Kube state metrics versions 2.4.2-2.7.0

## Resources

- [State container metrics](https://github.com/kubernetes/kube-state-metrics/blob/main/internal/store/pod.go):
declaration and description

## Metrics insight

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
- kube_pod_container_info
  - container
  - image_spec
  - image
  - image_id
  - container_id
- kube_pod_container_resource_requests
  - container
  - node
  - resource
  - unit
- kube_pod_container_resource_limits
  - container
  - node
  - resource
  - unit
- kube_pod_container_status_ready
  - container
- kube_pod_container_status_restarts_total
  - container
- kube_pod_container_status_running
  - container
- kube_pod_container_status_terminated_reason
  - container
  - reason
- kube_pod_container_status_waiting_reason
  - container
  - reason
- kube_pod_container_status_last_terminated_reason
  - container
  - reason


### Setup environment for manual tests
Go to `metricbeat/module/kubernetes/_meta/test/docs`.
