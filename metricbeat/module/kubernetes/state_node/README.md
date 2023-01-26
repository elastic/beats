### Version history

**January 2023**: Kube state metrics versions 2.4.2-2.7.0

### Resources

- [State node metrics](https://github.com/kubernetes/kube-state-metrics/blob/main/internal/store/node.go):
declaration and description

### Metrics insight

All metrics have the label:
- node

Additionally:

- kube_node_info
  - kernel_version
  - os_image
  - container_runtime_version
  - kubelet_version
  - kubeproxy_version
  - provider_id
  - pod_cidr
  - system_uuid
- kube_node_status_capacity
  - resource
  - unit
- kube_node_status_allocatable
  - resource
  - unit
- kube_node_spec_unschedulable
- kube_node_status_condition
  - condition
  - status



### Setup environment for manual tests
Go to `metricbeat/module/kubernetes/_meta/test/docs`.
