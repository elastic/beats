### Version history

**January 2023**: Kube state metrics versions 2.4.2-2.7.0

### Resources

- [State persistent volume claim metrics](https://github.com/kubernetes/kube-state-metrics/blob/main/internal/store/persistentvolumeclaim.go):
  declaration and description

### Metrics insight

All metrics have the label:
- namespace
- persistentvolumeclaim

Additionally:
- kube_persistentvolumeclaim_access_mode
  - access_mode
- kube_persistentvolumeclaim_labels
- kube_persistentvolumeclaim_info
  - storageclass
  - volumename
- kube_persistentvolumeclaim_resource_requests_storage_bytes
- kube_persistentvolumeclaim_status_phase
  - phase


### Setup environment for manual tests
Go to `metricbeat/module/kubernetes/_meta/test/docs`.

