---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-benchmark.html
  # That link will 404 until 8.18 is current
  # (see https://www.elastic.co/guide/en/beats/metricbeat/8.18/metricbeat-module-benchmark.html)
---

# Benchmark module [metricbeat-module-benchmark]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://manage-data/ingest/tools.md).

::::


:::::


The `benchmark` module is used to generate synthetic metrics at a predictable rate.  This can be useful when you want to test output settings or test system sizing without using real data.

The `benchmark` module metricset is `info`.

```yaml
- module: benchmark
  metricsets:
    - info
  enabled: true
  period: 10s
```


## Metricsets [_metricsets_13]


### `info` [_info_2]

A metric that includes a `counter` field which is used to keep the metric unique.


### Module-specific configuration notes [_module_specific_configuration_notes_3]

`count`
:   number, the number of metrics to emit per fetch.


### Example configuration [_example_configuration_9]

The Benchmark module supports the standard configuration options that are described in [Modules](configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: benchmark
  metricsets:
    - info
  enabled: false
  period: 10s
```


### Metricsets [_metricsets_14]

The following metricsets are available:

* [info](metricbeat-metricset-benchmark-info.md)


