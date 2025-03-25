---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-istio-proxy.html
---

# Istio proxy metricset [metricbeat-metricset-istio-proxy]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This is the proxy metricset of the module istio. This metricset collects metrics from the Envoy proxy’s Prometheus exporter for Istio versions higher than 1.5

Tested with Istio 1.7


## Deployment [_deployment]

Istio-proxy is a sidecar container that is being injected into every Pod that is being deployed on a Kubernetes cluster which’s traffic is managed by Istio. Because of this reason, in order to collect metrics from this sidecars we need to automatically identify these sidecar containers and start monitoring them using their IP and the predifined port (15090). This can be achieved easily by defining the proper autodiscover provider that will automatically identify all these sidecar containers and will start the `proxy` metricset for each one of them. Here is an example configuration that can be used for that purpose:

```yaml
metricbeat.autodiscover:
  providers:
    - type: kubernetes
      node: ${NODE_NAME}
      templates:
        - condition:
            contains:
              kubernetes.annotations.prometheus.io/path: "/stats/prometheus"
          config:
            - module: istio
              metricsets: ["proxy"]
              hosts: "${data.kubernetes.pod.ip}:15090"
```

## Fields [_fields_126]

For a description of each field in the metricset, see the [exported fields](/reference/metricbeat/exported-fields-istio.md) section.


