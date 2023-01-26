## Version history

**January 2023**: Kube state metrics versions 2.4.2-2.7.0

## Resources

- [Statefulset metrics](https://github.com/kubernetes/kube-state-metrics/blob/main/internal/store/statefulset.go):
  declaration and description

## Metrics insight

All metrics have the labels:
- namespace
- statefulset

Additionally:
- kube_statefulset_created
- kube_statefulset_metadata_generation
- kube_statefulset_status_observed_generation
- kube_statefulset_replicas
- kube_statefulset_status_replicas
- kube_statefulset_status_replicas_ready

### Setup environment for manual tests
Go to `metricbeat/module/kubernetes/_meta/test/docs`.
