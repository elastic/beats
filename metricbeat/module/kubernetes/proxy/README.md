# Kubeproxy Stats

## Version history

- June 2019, `v1.14.0`

## Resources

- https://github.com/kubernetes/kubernetes/blob/master/pkg/proxy/metrics/metrics.go
- https://kubernetes.io/docs/reference/command-line-tools-reference/kube-proxy/

## Metrics insight

Process metrics:
- process_cpu_seconds_total
- process_resident_memory_bytes
- process_virtual_memory_bytes

Network rules syncing metrics:
kubeproxy_sync_proxy_rules_duration_seconds_bucket
    - le

HTTP server metrics:
http_request_duration_microseconds
    - handler
    - quantile

Rest client metrics:
rest_client_request_duration_seconds_bucket
    - url
    - verb
    - le
rest_client_requests_total
    - code
    - host
    - method

## Setup environment for manual tests

- Create a kubernetes cluster
- Deploy metricbeat as a daemonset + host network
