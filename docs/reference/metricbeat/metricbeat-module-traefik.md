---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-traefik.html
---

# Traefik module [metricbeat-module-traefik]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/traefik/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This module periodically fetches metrics from a [Traefik](https://traefik.io/) instance. The Traefik instance must be configured to expose it’s HTTP API.


## Compatibility [_compatibility_49]

The Traefik metricsets were tested with Traefik 1.6.


## Example configuration [_example_configuration_65]

The Traefik module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: traefik
  metricsets: ["health"]
  period: 10s
  hosts: ["localhost:8080"]
```

This module supports TLS connections when using `ssl` config field, as described in [SSL](/reference/metricbeat/configuration-ssl.md). It also supports the options described in [Standard HTTP config options](/reference/metricbeat/configuration-metricbeat.md#module-http-config-options).


## Metricsets [_metricsets_75]

The following metricsets are available:

* [health](/reference/metricbeat/metricbeat-metricset-traefik-health.md)


