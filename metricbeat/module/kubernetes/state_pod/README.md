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

# Running integration tests.


Running the integration tests for the kubernetes module has the requirement of:

* docker
* kind
* kubectl

Once those tools are installed its as simple as:

```
MODULE="kubernetes" mage goIntegTest
```

The integration tester will use the default context from the kubectl configuration defined
in the `KUBECONFIG` environment variable. There is no requirement that the kubernetes even
be local to your development machine, it just needs to be accessible.

If no `KUBECONFIG` is set and `kind` is installed then the runner will use `kind` to create
a local cluster inside of your local docker to perform the intergation tests inside. The
`kind` cluster will be created and destroy before and after the test. If you would like to
keep the `kind` cluster running after the test has finished you can set `KIND_SKIP_DELETE=1`
inside of your environment.


## Starting Kubernetes clusters in Cloud providers

The `terraform` directory contains terraform configurations to start Kubernetes
clusters in cloud providers.
