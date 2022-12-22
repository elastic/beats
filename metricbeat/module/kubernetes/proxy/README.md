# Kubeproxy Stats

## Version history

- December 2022, `v1.19`

## Resources

- https://github.com/kubernetes/kubernetes/blob/master/pkg/proxy/metrics/metrics.go
- https://kubernetes.io/docs/reference/command-line-tools-reference/kube-proxy/
- https://kubernetes.io/docs/reference/instrumentation/metrics/

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
