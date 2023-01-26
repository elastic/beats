### Version history

**January 2023**: Kube state metrics versions 2.4.2-2.7.0

### Resources

- [State replicaset metrics](https://github.com/kubernetes/kube-state-metrics/blob/main/internal/store/replicaset.go):
declaration and description

### Metrics insight

All metrics have the labels:
- replicaset
- namespace

Additionally:
- kube_replicaset_metadata_generation
- kube_replicaset_status_fully_labeled_replicas
- kube_replicaset_status_observed_generation
- kube_replicaset_status_ready_replicas
- kube_replicaset_spec_replicas
- kube_replicaset_status_replicas

### Setup environment for manual tests
Go to `metricbeat/module/kubernetes/_meta/test/docs`.

