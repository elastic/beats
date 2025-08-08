---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-windows.html
---

% This file is generated! See scripts/docs_collector.py

# Windows module [metricbeat-module-windows]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/windows/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This is the `windows` module which collects metrics from Windows systems. The module contains the `service` metricset, which is set up by default when the `windows` module is enabled. The `service` metricset will retrieve status information of the services on the Windows machines. The second `windows` metricset is `perfmon` which collects Windows performance counter values.


## Example configuration [_example_configuration]

The Windows module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: windows
  metricsets: ["perfmon"]
  enabled: true
  period: 10s
  perfmon.ignore_non_existent_counters: false
  perfmon.group_measurements_by_instance: false
  perfmon.queries:
#  - object: 'Process'
#    instance: ["*"]
#    counters:
#    - name: '% Processor Time'
#      field: cpu_usage
#      format: "float"
#    - name: "Thread Count"

- module: windows
  metricsets: ["service"]
  enabled: true
  period: 60s
```


## Metricsets [_metricsets]

The following metricsets are available:

* [perfmon](/reference/metricbeat/metricbeat-metricset-windows-perfmon.md)
* [service](/reference/metricbeat/metricbeat-metricset-windows-service.md)
<<<<<<< HEAD
=======
* [wmi](/reference/metricbeat/metricbeat-metricset-windows-wmi.md)  {applies_to}`stack: beta`
>>>>>>> f955bca20 ([docs] [metricbeat] Update `docs_collector.go` to add `applies_to` badges (#45837))
