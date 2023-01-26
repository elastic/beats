# Kube-state-metrics/Cronjob

### Version history

**January 2023**: Kube state metrics versions 2.4.2-2.7.0

### Resources

- [State daemonset metrics](https://github.com/kubernetes/kube-state-metrics/blob/main/internal/store/daemonset.go):
  declaration and description

### Metrics insight

All metrics have the labels:
- daemonset
- namespace

Metrics:
- kube_daemonset_metadata_generation
- kube_daemonset_status_number_available
- kube_daemonset_status_desired_number_scheduled
- kube_daemonset_status_number_ready


### Setup environment for manual tests
Go to `metricbeat/module/kubernetes/_meta/test/docs`.
