---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-kibana.html
---

# Kibana module [metricbeat-module-kibana]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/kibana/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The `kibana` module collects metrics about {{kib}}.


## Compatibility [_compatibility_30]

The `kibana` module works with {{kib}} 6.7.0 and later.


## Usage for {{stack}} Monitoring [_usage_for_stack_monitoring_4]

The `kibana` module can be used to collect metrics shown in our {{stack-monitor-app}} UI in {{kib}}. To enable this usage, set `xpack.enabled: true` and remove any `metricsets` from the module’s configuration. Alternatively, run `metricbeat modules disable kibana` and `metricbeat modules enable kibana-xpack`.

::::{note}
When this module is used for {{stack}} Monitoring, it sends metrics to the monitoring index instead of the default index typically used by {{metricbeat}}. For more details about the monitoring index, see [Configuring indices for monitoring](docs-content://deploy-manage/monitor/monitoring-data/configuring-data-streamsindices-for-monitoring.md).
::::



## Example configuration [_example_configuration_36]

The Kibana module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: kibana
  metricsets: ["status"]
  period: 10s
  hosts: ["localhost:5601"]
  basepath: ""
  enabled: true
  #username: "user"
  #password: "secret"
  #api_key: "foo:bar"

  # Set to true to send data collected by module to X-Pack
  # Monitoring instead of metricbeat-* indices.
  #xpack.enabled: false
```

This module supports TLS connections when using `ssl` config field, as described in [SSL](/reference/metricbeat/configuration-ssl.md). It also supports the options described in [Standard HTTP config options](/reference/metricbeat/configuration-metricbeat.md#module-http-config-options).


## Metricsets [_metricsets_42]

The following metricsets are available:

* [cluster_actions](/reference/metricbeat/metricbeat-metricset-kibana-cluster_actions.md)
* [cluster_rules](/reference/metricbeat/metricbeat-metricset-kibana-cluster_rules.md)
* [node_actions](/reference/metricbeat/metricbeat-metricset-kibana-node_actions.md)
* [node_rules](/reference/metricbeat/metricbeat-metricset-kibana-node_rules.md)
* [stats](/reference/metricbeat/metricbeat-metricset-kibana-stats.md)
* [status](/reference/metricbeat/metricbeat-metricset-kibana-status.md)








