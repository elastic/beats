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


## Setup environment for manual tests


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
