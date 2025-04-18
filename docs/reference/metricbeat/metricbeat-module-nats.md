---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-nats.html
---

# NATS module [metricbeat-module-nats]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/nats/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The Nats module uses [Nats monitoring server APIs](https://docs.nats.io/running-a-nats-service/nats_admin/monitoring) to collect metrics.

The default metricsets are `stats`, `connections`, `routes` and `subscriptions`. The `connection`, `route`, and `jetstream` metricsets can be enabled to collect additional metrics.

## Compatibility [_compatibility_39]

The NATS module is tested with NATS 2.2.6 and 2.11.x. Versions in between are expected to be compatible as well.


## Dashboard [_dashboard_34]

The Nats module comes with a predefined dashboard. For example:

![metricbeat nats dashboard](images/metricbeat_nats_dashboard.png)


## Example configuration [_example_configuration_47]

The NATS module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: nats
  metricsets:
    - "connections"
    - "routes"
    - "stats"
    - "subscriptions"
    #- "connection"
    #- "route"
    #- "jetstream"
  period: 10s
  hosts: ["localhost:8222"]
  #stats.metrics_path: "/varz"
  #connections.metrics_path: "/connz"
  #routes.metrics_path: "/routez"
  #subscriptions.metrics_path: "/subsz"
  #connection.metrics_path: "/connz"
  #route.metrics_path: "/routez"
  #jetstream:
  #  stats:
  #    enabled: true
  #  account:
  #    enabled: true
  #    names:
  #      - default
  #  stream:
  #    enabled: true
  #    names:
  #      - my-stream-1
  #      - another-stream
  #  consumer:
  #    enabled: true
  #    names:
  #      - my-stream-1-consumer-1
  #      - my-stream-1-consumer-2
  #      - another-stream-consumer-1
```

This module supports TLS connections when using `ssl` config field, as described in [SSL](/reference/metricbeat/configuration-ssl.md). It also supports the options described in [Standard HTTP config options](/reference/metricbeat/configuration-metricbeat.md#module-http-config-options).


## Metricsets [_metricsets_54]

The following metricsets are available:

* [connection](/reference/metricbeat/metricbeat-metricset-nats-connection.md)
* [connections](/reference/metricbeat/metricbeat-metricset-nats-connections.md)
* [jetstream](/reference/metricbeat/metricbeat-metricset-nats-jetstream.md)
* [route](/reference/metricbeat/metricbeat-metricset-nats-route.md)
* [routes](/reference/metricbeat/metricbeat-metricset-nats-routes.md)
* [stats](/reference/metricbeat/metricbeat-metricset-nats-stats.md)
* [subscriptions](/reference/metricbeat/metricbeat-metricset-nats-subscriptions.md)







