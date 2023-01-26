# Kube-state-metrics/Cronjob

### Version history

**January 2023**: Kube state metrics versions 2.4.2-2.7.0

### Resources

- [State deployment metrics](https://github.com/kubernetes/kube-state-metrics/blob/main/internal/store/deployment.go):
  declaration and description

### Metrics insight

All metrics have the labels:
- deployment
- namespace

Metrics:
- kube_deployment_metadata_generation
- kube_deployment_status_replicas_updated
- kube_deployment_status_replicas_unavailable
- kube_deployment_status_replicas_available
- kube_deployment_spec_replicas

### Setup environment for manual tests
Go to `metricbeat/module/kubernetes/_meta/test/docs`.
