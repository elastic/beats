### Version history

**January 2023**: Kube state metrics versions 2.4.2-2.7.0

### Resources

- [State storage class metrics](https://github.com/kubernetes/kube-state-metrics/blob/main/internal/store/storageclass.go):
  declaration and description

### Metrics insight

All metrics have the labels:
- namespace
- pod
- uid

Additionally:
- kube_storageclass_info
  - provisioner
  - reclaim_policy
  - volume_binding_mode
- kube_storageclass_labels
- kube_storageclass_created

### Setup environment for manual tests
Go to `metricbeat/module/kubernetes/_meta/test/docs`.
