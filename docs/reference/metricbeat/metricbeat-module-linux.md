---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-linux.html
---

# Linux module [metricbeat-module-linux]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/linux/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The Linux module reports on metrics exclusive to the Linux kernel and GNU/Linux OS.


## Example configuration [_example_configuration_39]

The Linux module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: linux
  period: 10s
  metricsets:
    - "pageinfo"
    - "memory"
    # - ksm
    # - conntrack
    # - iostat
    # - pressure
    # - rapl
  enabled: true
  #hostfs: /hostfs
  #rapl.use_msr_safe: false
```


## Metricsets [_metricsets_45]

The following metricsets are available:

* [conntrack](/reference/metricbeat/metricbeat-metricset-linux-conntrack.md)
* [iostat](/reference/metricbeat/metricbeat-metricset-linux-iostat.md)
* [ksm](/reference/metricbeat/metricbeat-metricset-linux-ksm.md)
* [memory](/reference/metricbeat/metricbeat-metricset-linux-memory.md)
* [pageinfo](/reference/metricbeat/metricbeat-metricset-linux-pageinfo.md)
* [pressure](/reference/metricbeat/metricbeat-metricset-linux-pressure.md)
* [rapl](/reference/metricbeat/metricbeat-metricset-linux-rapl.md)








