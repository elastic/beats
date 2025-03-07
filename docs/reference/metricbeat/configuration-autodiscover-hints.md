---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/configuration-autodiscover-hints.html
---

# Hints based autodiscover [configuration-autodiscover-hints]

Metricbeat supports autodiscover based on hints from the provider. The `hints` system looks for hints in Kubernetes Pod annotations or Docker labels which have the prefix `co.elastic.metrics`. As soon as the container starts, Metricbeat will check if it contains any hints and launch the proper config for it. Hints tell Metricbeat how to get metrics for the given container. This is the full list of supported hints:


#### `co.elastic.metrics/module` [_co_elastic_metricsmodule]

Metricbeat module to use to fetch metrics. See [Modules](/reference/metricbeat/metricbeat-modules.md) for the list of supported modules.


#### `co.elastic.metrics/hosts` [_co_elastic_metricshosts]

Hosts setting to use with the given module. Hosts can include `${data.host}`, `${data.port}`, `${data.ports.<port_name>}` values from the autodiscover event, ie: `${data.host}:80`. For Kuberentes autodiscover events users can leverage port names as well, like `http://${data.host}:${data.ports.prometheus}/metrics`. In the last one we refer to the container port with name `prometheus`.

In order to target one specific exposed port `localhost:{data.ports.8222}` can be used in template config, where `8222` is the internal container port of the exposed targeted port.


#### `co.elastic.metrics/metricsets` [_co_elastic_metricsmetricsets]

List of metricsets to use, comma separated. If no metricsets are provided, default metricsets for the module are used.


#### `co.elastic.metrics/metrics_path` [_co_elastic_metricsmetrics_path]

The path to retrieve the metrics from (/metrics by default) for [Prometheus module](/reference/metricbeat/metricbeat-module-prometheus.md#prometheus-module).


#### `co.elastic.metrics/period` [_co_elastic_metricsperiod]

The time interval for metrics retrieval, ie: 10s


#### `co.elastic.metrics/timeout` [_co_elastic_metricstimeout]

Metrics retrieval timeout, default: 3s


#### `co.elastic.metrics/username` [_co_elastic_metricsusername]

The username to use for authentication


#### `co.elastic.metrics/password` [_co_elastic_metricspassword]

The password to use for authentication. It is recommended to retrieve this sensitive information from an ENV variable and avoid placing passwords in plain text. Unlike static autodiscover configuration, hints based autodiscover has no access to the keystore of Metricbeat since it could be a potential security issue. However hints based autodiscover can make use of Kuberentes Secrets as described in [Metricbeat Autodiscover Secret Management](/reference/metricbeat/configuration-autodiscover.md#kubernetes-secrets).


#### `co.elastic.metrics/ssl.*` [_co_elastic_metricsssl]

SSL parameters, as seen in [SSL](/reference/metricbeat/configuration-ssl.md).


#### `co.elastic.metrics/metrics_filters.*` [_co_elastic_metricsmetrics_filters]

Metrics filters (for prometheus module only).

```yaml
co.elastic.metrics/module: prometheus
co.elastic.metrics/metrics_filters.include: node_filesystem_*
co.elastic.metrics/metrics_filters.exclude: node_filesystem_device_foo,node_filesystem_device_bar
```


#### `co.elastic.metrics/raw` [_co_elastic_metricsraw]

When an entire module configuration needs to be completely set the `raw` hint can be used. You can provide a stringified JSON of the input configuration. `raw` overrides every other hint and can be used to create both a single or a list of configurations.

```yaml
co.elastic.metrics/raw: "[{\"enabled\":true,\"metricsets\":[\"default\"],\"module\":\"mockmoduledefaults\",\"period\":\"1m\",\"timeout\":\"3s\"}]"
```


#### `co.elastic.metrics/processors` [_co_elastic_metricsprocessors]

Define a processor to be added to the Metricbeat module configuration. See [Processors](/reference/metricbeat/filtering-enhancing-data.md) for the list of supported processors.

In order to provide ordering of the processor definition, numbers can be provided. If not, the hints builder will do arbitrary ordering:

```yaml
co.elastic.logs/processors.1.add_locale.abbrevation: "MST"
co.elastic.logs/processors.add_locale.abbrevation: "PST"
```

In the above sample the processor definition tagged with `1` would be executed first.

When hints are used along with templates, then hints will be evaluated only in case there is no template’s condition that resolves to true. For example:

```yaml
metricbeat.autodiscover.providers:
  - type: kubernetes
    hints.enabled: true
    templates:
        - condition:
            contains:
              kubernetes.annotations.prometheus.io/scrape: "true"
          config:
            - module: prometheus
              metricsets: ["collector"]
              hosts: "${data.host}:${data.port}"
```

In this example first the condition `kubernetes.annotations.prometheus.io/scrape: "true"` is evaluated and if not matched the hints will be processed.


## Kubernetes [_kubernetes_2]

Kubernetes autodiscover provider supports hints in Pod annotations. To enable it just set `hints.enabled`:

```yaml
metricbeat.autodiscover:
  providers:
    - type: kubernetes
      hints.enabled: true
```

This configuration enables the `hints` autodiscover for Kubernetes. The `hints` system looks for hints in Kubernetes annotations or Docker labels which have the prefix `co.elastic.metrics`.

You can annotate Kubernetes Pods with useful info to spin up Metricbeat modules:

```yaml
annotations:
  co.elastic.metrics/module: prometheus
  co.elastic.metrics/metricsets: collector
  co.elastic.metrics/hosts: '${data.host}:9090'
  co.elastic.metrics/period: 1m
```

The above annotations are used to spin up a Prometheus collector metricset and it polls the parent container on port `9090` at a 1 minute interval.


#### Multiple containers [_multiple_containers]

When a Pod has multiple containers, these settings are shared. To set hints specific to a container in the pod you can put the container name in the hint.

When a pod has multiple containers, the settings are shared unless you put the container name in the hint. For example, these hints configure a common behavior for all containers in the Pod, and sets a specific `hosts` hint for the container called `sidecar`.

```yaml
annotations:
  co.elastic.metrics/module: apache
  co.elastic.metrics/hosts: '${data.host}:80'
  co.elastic.metrics.sidecar/hosts: '${data.host}:8080'
```


#### Multiple sets of hints [_multiple_sets_of_hints]

When a container port needs multiple modules to be defined on it, sets of annotations can be provided with numeric prefixes. If there are hints that don’t have a numeric prefix then they get grouped together into a single configuration.

```yaml
annotations:
  co.elastic.metrics/1.module: prometheus
  co.elastic.metrics/1.hosts: '${data.host}:80/metrics'
  co.elastic.metrics/1.period: 60s
  co.elastic.metrics/module: prometheus
  co.elastic.metrics/hosts: '${data.host}:80/metrics/p1'
  co.elastic.metrics/period: 5s
```

The above configuration would spin up two metricbeat module configurations to ensure that the endpoint "/metrics/p1" is polled every 5s whereas the "/metrics" endpoint is polled every 60s.


#### Namespace Defaults [_namespace_defaults]

Hints can be configured on the Namespace’s annotations as defaults to use when Pod level annotations are missing. The resultant hints are a combination of Pod annotations and Namespace annotations with the Pod’s taking precedence. To enable Namespace defaults configure the `add_resource_metadata` for Namespace objects as follows:

```yaml
metricbeat.autodiscover:
  providers:
    - type: kubernetes
      hints.enabled: true
      add_resource_metadata:
        namespace:
          include_annotations: ["nsannotation1"]
```


## Docker [_docker_3]

Docker autodiscover provider supports hints in labels. To enable it just set `hints.enabled`:

```yaml
metricbeat.autodiscover:
  providers:
    - type: docker
      hints.enabled: true
```

You can label Docker containers with useful info to spin up Metricbeat modules, for example:

```yaml
  co.elastic.metrics/module: nginx
  co.elastic.metrics/metricsets: stubstatus
  co.elastic.metrics/hosts: '${data.host}:80'
  co.elastic.metrics/period: 10s
```

The above labels would allow Metricbeat to run the nginx module and poll port `80` of the Docker container every 10 seconds.

