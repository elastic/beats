---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-windows.html
---

# Windows module [metricbeat-module-windows]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/windows.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/ingestion-tools/fleet/index.md).

::::


:::::


This is the `windows` module which collects metrics from Windows systems. The module contains the `service` metricset, which is set up by default when the `windows` module is enabled. The `service` metricset will retrieve status information of the services on the Windows machines. The second `windows` metricset is `perfmon` which collects Windows performance counter values.


## Example configuration [_example_configuration_68]

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
<<<<<<< HEAD:metricbeat/docs/modules/windows.asciidoc
----
=======

- module: windows
  metricsets: ["wmi"]
  period: 60s
  wmi:
    include_null: false          # Exclude fields with null values from the output
    include_queries: false       # Do not include the query string in the output
    include_empty_string: false  # Exclude fields with empty string values from the output
    warning_threshold: 30s       # Maximum time to wait for a query result before logging a warning (defaults to period)
    # Default WMI namespace for all queries (if not specified per query)
    # Uncomment to override the default, which is "root\\cimv2".
    # namespace: "root\\cimv2"
    queries:
    - class: Win32_OperatingSystem # FROM: Class to fetch
      fields:                      # SELECT: Fields to retrieve for this WMI class. Omit the setting to fetch all properties
       - FreePhysicalMemory
       - FreeSpaceInPagingFiles
       - FreeVirtualMemory
       - LocalDateTime
       - NumberOfUsers
      where: ""                   # Optional WHERE clause to filter query results
      # Override the WMI namespace for this specific query (optional).
      # If set, this takes precedence over the default namespace above.
      # namespace: "root\\cimv2" # Overrides the metric
```
>>>>>>> 3102f5eba ([docs] Migrate docs from AsciiDoc to Markdown (#42897)):docs/reference/metricbeat/metricbeat-module-windows.md


## Metricsets [_metricsets_78]

The following metricsets are available:

* [perfmon](/reference/metricbeat/metricbeat-metricset-windows-perfmon.md)
* [service](/reference/metricbeat/metricbeat-metricset-windows-service.md)
* [wmi](/reference/metricbeat/metricbeat-metricset-windows-wmi.md)


<<<<<<< HEAD:metricbeat/docs/modules/windows.asciidoc
include::windows/perfmon.asciidoc[]
=======

>>>>>>> 3102f5eba ([docs] Migrate docs from AsciiDoc to Markdown (#42897)):docs/reference/metricbeat/metricbeat-module-windows.md


<<<<<<< HEAD:metricbeat/docs/modules/windows.asciidoc
:edit_url!:
=======

>>>>>>> 3102f5eba ([docs] Migrate docs from AsciiDoc to Markdown (#42897)):docs/reference/metricbeat/metricbeat-module-windows.md
