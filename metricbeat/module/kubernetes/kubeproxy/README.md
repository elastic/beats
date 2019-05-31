# Kubeproxy Stats

## Version history

- June 2019, `v1.13.0`

## Resources

- https://github.com/kubernetes/kubernetes/blob/master/pkg/proxy/metrics/metrics.go
- https://kubernetes.io/docs/reference/command-line-tools-reference/kube-proxy/

## Metrics insight

process_cpu_seconds_total
process_resident_memory_bytes
process_virtual_memory_bytes

kubeproxy_sync_proxy_rules_latency_microseconds_bucket
    - le

http_request_duration_microseconds
    - handler
    - quantile

rest_client_request_latency_seconds_bucket (integer)
    - url
    - verb
    - le

rest_client_requests_total (integer)
    - code
    - host
    - method

## Setup environment for manual tests

- Create a kubernetes cluster
- Deploy metricbeat as a daemonset + host network













