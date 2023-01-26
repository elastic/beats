### Version history

**January 2023**: Kube state metrics versions 2.4.2-2.7.0

### Resources

- [State service metrics](https://github.com/kubernetes/kube-state-metrics/blob/main/internal/store/service.go):
  declaration and description

### Metrics insight

All metrics have the labels:
- namespace
- uid
- service

Additionally:
- kube_service_info
  - cluster_ip
  - external_name
  - load_balancer_ip
- kube_service_created
- kube_service_spec_type
  - type
- kube_service_spec_external_ip
  - external_ip
- kube_service_status_load_balancer_ingress
  - ip
  - hostname

### Setup environment for manual tests
Go to `metricbeat/module/kubernetes/_meta/test/docs`.
