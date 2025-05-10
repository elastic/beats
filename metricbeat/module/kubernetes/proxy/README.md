# Kubeproxy Stats

## Version history

- December 2022, `v1.25.x`

## Resources

- [Process metrics](https://github.com/kubernetes/kubernetes/blob/master/vendor/github.com/prometheus/client_golang/prometheus/process_collector.go)
- [Proxy metrics](https://github.com/kubernetes/kubernetes/blob/master/pkg/proxy/metrics/metrics.go)
- [Rest client metrics](https://github.com/kubernetes/component-base/blob/master/metrics/prometheus/restclient/metrics.go)
- [Metrics general information](https://kubernetes.io/docs/reference/instrumentation/metrics/)


## Metrics insight

- process_cpu_seconds_total
- process_resident_memory_bytes
- process_virtual_memory_bytes
- process_open_fds
- process_start_time_seconds
- process_max_fds


- rest_client_requests_total (alpha)
    - code
    - host
    - method
- rest_client_response_size_bytes (alpha)
    - host
    - verb
- rest_client_request_size_bytes (alpha)
    - host
    - verb
- rest_client_request_duration_seconds (alpha)
    - host
    - verb


- kubeproxy_sync_proxy_rules_duration_seconds (alpha)
- kubeproxy_network_programming_duration_seconds (alpha)

## Setup environment for manual tests

- Create a kubernetes cluster
- Deploy metricbeat as a daemonset + host network

## Generating expectation files

In order to support a new Kubernetes releases you'll have to generate new expectation files for this module in `_meta/test`. For that, start by deploying a new kubernetes cluster on the required Kubernetes version, for example:

```bash
kind create cluster --image kindest/node:v1.32.0
```

After that, you can fetch the proxy metrics from the api:

```bash
kubectl proxy
```

Then you can fetch the metrics from the url provided in the output and save it to a new `_meta/test/metrics.x.xx` file.

After that, you can run the following commands to generate and test the expected files:

```bash
cd metricbeat/module/kubernetes/proxy
# generate the expected files
go test ./state... --data
# test the expected files
go test ./state...
```
