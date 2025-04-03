---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-logstash.html
---

# Logstash module [metricbeat-module-logstash]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/logstash/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The `logstash` module collects metrics about {{ls}}.


## Compatibility [_compatibility_32]

The `logstash` module works with {{ls}} 7.3.0 and later.


## Usage for Stack Monitoring [_usage_for_stack_monitoring_5]

The `logstash` module can be used to collect metrics shown in our {{stack-monitor-app}} UI in {{kib}}. To enable this usage, set `xpack.enabled: true` and remove any `metricsets` from the module’s configuration. Alternatively, run `metricbeat modules disable logstash` and `metricbeat modules enable logstash-xpack`.

::::{note}
When this module is used for {{stack}} Monitoring, it sends metrics to the monitoring index instead of the default index typically used by {{metricbeat}}. For more details about the monitoring index, see [Configuring indices for monitoring](docs-content://deploy-manage/monitor/monitoring-data/configuring-data-streamsindices-for-monitoring.md).
::::



## Example configuration [_example_configuration_40]

The Logstash module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: logstash
  metricsets: ["node", "node_stats"]
  enabled: true
  period: 10s
  hosts: ["localhost:9600"]
```

This module supports TLS connections when using `ssl` config field, as described in [SSL](/reference/metricbeat/configuration-ssl.md). It also supports the options described in [Standard HTTP config options](/reference/metricbeat/configuration-metricbeat.md#module-http-config-options).


## Metricsets [_metricsets_46]

The following metricsets are available:

* [node](/reference/metricbeat/metricbeat-metricset-logstash-node.md)
* [node_stats](/reference/metricbeat/metricbeat-metricset-logstash-node_stats.md)



