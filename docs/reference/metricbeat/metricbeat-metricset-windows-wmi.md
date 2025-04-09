---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/master/metricbeat-metricset-windows-wmi.html
---

# Windows wmi metricset [metricbeat-metricset-windows-wmi]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


The `wmi` metricset of the Windows module reads metrics via Windows Management Instrumentation  [(WMI)](https://learn.microsoft.com/en-us/windows/win32/wmisdk/about-wmi), a core management technology in the Windows Operating system.

By leveraging WMI Query Language (WQL), this metricset allows you to extract detailed system information and metrics to monitor the health and performance of Windows Systems.

This metricset leverages the [Microsoft WMI](https://github.com/microsoft/wmi), library a convenient wrapper around the [GO-OLE](https://github.com/go-ole) library which allows to invoke the WMI Api.

## WMI Query Language (WQL) Support

This metricset supports the execution of
[WQL](https://learn.microsoft.com/en-us/windows/win32/wmisdk/wql-sql-for-wmi)
queries, a SQL-like query language for retrieving information from WMI
namespaces.

Currently, the metricset supports queries with `SELECT`, `FROM` and
`WHERE` clauses.

When working with WMI queries, it is the user’s responsibility to ensure
that queries are safe, efficient, and do not cause unintended side
effects. A notorious example of a problematic WMI class is
Win32\_Product. Read more in [Windows
Documentation](https://support.microsoft.com/kb/974524).

## WMI Arbitrator and Query Execution

Query execution is managed by the underlying WMI Framework, specifically
the [WMI
Arbitrator](https://learn.microsoft.com/en-us/troubleshoot/windows-server/system-management-components/new-wmi-arbitrator-behavior-in-windows-server).
The Arbitrator is responsible for:

- Scheduling and controlling query execution

- Throttling or stopping queries based on system resource availability
  and conditions

There is no way to directly stop a query once it has started. To prevent
Metricbeat from waiting indefinitely for a query to return a result or
fail, Metricbeat has a timeout mechanism that stops waiting for query
results after a specified timeout. This is controlled by the
`wmi.warning_threshold` setting.

While Metricbeat stops waiting for the result, the underlying WMI query
may continue running until the WMI Arbitrator decides to stop execution.

## Configuration

```yaml
- module: windows
  metricsets: ["wmi"]
  period: 10m
  wmi:
    namespace: "root\\cimv2" # Default Namespace
    warning_threshold: 10m
    include_queries: true
    include_null_properties: false
    include_empty_strings_properties: false
    queries:
    - class: Win32_OperatingSystem
      properties:
      - FreePhysicalMemory
      - FreeSpaceInPagingFiles
      - NumberOfUsers
      where: ""
    - class: Win32_PowerPlan
      properties: []
      where: "IsActive = True"
      namespace: "root\\cimv2\\power" # Overwrites the module namespace in this query
```


**`wmi.namespace`**
:   The default WMI namespace used for queries. This can be overridden per
query. The default is `root\cimv2`.

**`wmi.warning_threshold`**
:   The time threshold after which Metricbeat will stop waiting for the
query result and return control to the main flow of the program. A
warning is logged indicating that the query execution has exceeded the
threshold. The default is equal to the period. See [WMI Arbitrator and
Query Execution](#wmi-arbitrator-and-query-execution) for more details.

**`wmi.include_queries`**
:   If set to `true` the metricset includes the query in the output
document. The default value is `false`.

**`wmi.include_null_properties`**
:   If set to `true` the metricset includes the properties that have null
value in the output document. properties that have a `null` value in the
output document. The default value is `false`.

**`wmi.include_empty_string_properties`**
:   A boolean option that causes the metricset to include the properties
that are empty string. The default value is `false`.

**`wmi.queries`**
:   The list of queries to execute. The list cannot be empty. See [Query
Configuration](#query-configuration) for the format of the queries.

### Query Configuration

Each item in the `queries` list specifies a wmi query to perform.

**`class`**
:    The wmi class. In the query it specifies the `FROM` clause. Required

**`properties`**
:    List of properties to return. In the query it specifies the `SELECT`
clause. Set it to the empty list (default value) to retrieve all
available properties.

**`where`**
:   The where clause. In the query it specifies the `WHERE` clause. Read
more about the format [in the Windows
Documentation](https://learn.microsoft.com/en-us/windows/win32/wmisdk/where-clause).

**`namespace`**
:   The WMI Namespace for this particular query (it overwrites the
metricset’s `namespace` value)

### Example

Example WQL Query:

```sql
SELECT Name, ProcessId, WorkingSetSize
FROM Win32_Process
WHERE Name = 'lsass.exe' AND WorkingSetSize > 104857600
```

Equivalent YAML Configuration:

```yaml
- class: Win32_Process
  properties:
  - Name
  - ProcessId
  - WorkingSetSize
  where: "Name = 'lsass.exe' AND WorkingSetSize > 104857600"
```

## Best Practices

- Test your queries in isolation using the `Get-CimInstance` PowerShell
  cmdlet or the WMI Explorer.

- Ensure that `wmi.warning_threshold` is **less than or equal to** the
  module’s `period`. This prevents starting intentionally multiple
  executions of the same query.

- Set up alerts in Metricbeat logs for timeouts and empty query results.
  If a query frequently times out or returns no data, investigate the
  cause to prevent missing critical information.

- \[Advanced\] Collect WMI-Activity Operational Logs to correlate with
  Metricbeat WMI warnings.

## Compatibility

This module has been tested on the following platform:

- Operating System: Microsoft Windows Server 2019 Datacenter

- Architecture: x64

Other Windows versions and architectures may also work but have not been
explicitly tested.


