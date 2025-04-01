---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-elasticsearch.html
---

# Elasticsearch module [metricbeat-module-elasticsearch]

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


## Example configuration [_example_configuration_20]

The Elasticsearch module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: elasticsearch
  metricsets:
    - node
    - node_stats
    #- index
    #- index_recovery
    #- index_summary
    #- ingest_pipeline
    #- shard
    #- ml_job
  period: 10s
  hosts: ["http://localhost:9200"]
  #username: "elastic"
  #password: "changeme"
  #api_key: "foo:bar"
  #ssl.certificate_authorities: ["/etc/pki/root/ca.pem"]

  #index_recovery.active_only: true
  #ingest_pipeline.processor_sample_rate: 0.25
  #xpack.enabled: false
  #scope: node
```

This module supports TLS connections when using `ssl` config field, as described in [SSL](/reference/metricbeat/configuration-ssl.md). It also supports the options described in [Standard HTTP config options](/reference/metricbeat/configuration-metricbeat.md#module-http-config-options).


## Metricsets [_metricsets_26]

The following metricsets are available:

* [ccr](/reference/metricbeat/metricbeat-metricset-elasticsearch-ccr.md)
* [cluster_stats](/reference/metricbeat/metricbeat-metricset-elasticsearch-cluster_stats.md)
* [enrich](/reference/metricbeat/metricbeat-metricset-elasticsearch-enrich.md)
* [index](/reference/metricbeat/metricbeat-metricset-elasticsearch-index.md)
* [index_recovery](/reference/metricbeat/metricbeat-metricset-elasticsearch-index_recovery.md)
* [index_summary](/reference/metricbeat/metricbeat-metricset-elasticsearch-index_summary.md)
* [ingest_pipeline](/reference/metricbeat/metricbeat-metricset-elasticsearch-ingest_pipeline.md)
* [ml_job](/reference/metricbeat/metricbeat-metricset-elasticsearch-ml_job.md)
* [node](/reference/metricbeat/metricbeat-metricset-elasticsearch-node.md)
* [node_stats](/reference/metricbeat/metricbeat-metricset-elasticsearch-node_stats.md)
* [pending_tasks](/reference/metricbeat/metricbeat-metricset-elasticsearch-pending_tasks.md)
* [shard](/reference/metricbeat/metricbeat-metricset-elasticsearch-shard.md)













