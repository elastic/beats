This is the `apiserver` metricset of the Kubernetes module, in charge of retrieving metrics from the Kubernetes API (available at `/metrics`).

This metricset needs access to the `apiserver` component of Kubernetes, accessible typically by any POD via the `kubernetes.default` service or via environment variables (`KUBERNETES_SERVICE_HOST` and `KUBERNETES_SERVICE_PORT`).

When the API uses https, the pod will need to authenticate using its default token and trust the server using the appropiate CA file.

Configuration example using https and token based authentication:

```yaml
- module: kubernetes
  enabled: true
  metricsets:
    - apiserver
  hosts: ["https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}"]
  #hosts: ["https://kubernetes.default"]
  bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
  ssl.certificate_authorities:
    - /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
  period: 30s
```

In order to access the `/metrics` path of the API service, some Kubernetes environments might require the following permission to be added to a ClusterRole.

```yaml
rules:
- nonResourceURLs:
  - /metrics
  verbs:
  - get
```

The previous configuration and RBAC requirement is available in the complete example manifest proposed in [Running Metricbeat on Kubernetes](/reference/metricbeat/running-on-kubernetes.md) document.
