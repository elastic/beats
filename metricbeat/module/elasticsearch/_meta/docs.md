:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/elasticsearch/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The `elasticsearch` module collects metrics about {{es}}.


## Compatibility [_compatibility_18]

The `elasticsearch` module works with {{es}} 6.7.0 and later.


## Usage for {{stack}} Monitoring [_usage_for_stack_monitoring_2]

The `elasticsearch` module can be used to collect metrics shown in our {{stack-monitor-app}} UI in {{kib}}. To enable this usage, set `xpack.enabled: true` and remove any `metricsets` from the module’s configuration. Alternatively, run `metricbeat modules disable elasticsearch` and `metricbeat modules enable elasticsearch-xpack`.

When xpack mode is enabled, all the legacy metricsets are automatically enabled by default. This means that you do not need to manually enable them, and there will be no conflicts or issues because Metricbeat merges the user-defined metricsets with the ones that xpack mode forces to enable. As a result, you can seamlessly collect comprehensive metrics without worrying about dataset overlap or duplication.

```yaml
metricbeat.modules:
- module: elasticsearch
  xpack.enabled: true
  metricsets:
    - ingest_pipeline
  period: 10s
```

::::{note}
When this module is used for {{stack}} Monitoring, it sends metrics to the monitoring index instead of the default index typically used by {{metricbeat}}. For more details about the monitoring index, see [Configuring indices for monitoring](docs-content://deploy-manage/monitor/monitoring-data/configuring-data-streamsindices-for-monitoring.md).
::::



## Module-specific configuration notes [_module_specific_configuration_notes_7]

Like other Metricbeat modules, the `elasticsearch` module accepts a `hosts` configuration setting. This setting can contain a list of entries. The related `scope` setting determines how each entry in the `hosts` list is interpreted by the module.

* If `scope` is set to `node` (default), each entry in the `hosts` list indicates a distinct node in an {{es}} cluster.
* If `scope` is set to `cluster`, each entry in the `hosts` list indicates a single endpoint for a distinct {{es}} cluster (for example, a load-balancing proxy fronting the cluster).
