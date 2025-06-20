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
`Win32_Product`. Read more in [Windows
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

## WMI Type support

The `microsoft/wmi` library internally uses the **WMI Scripting API**. This API, as per the
[official WMI Documentation](https://learn.microsoft.com/en-us/windows/win32/wmisdk/querying-wmi),
does not provide direct type conversion for `uint64`, `sint64`, and `datetime` CIM types;
instead, these values are returned as strings.

To ensure the correct data type is reported, Metricbeat dynamically fetches the
CIM type definitions for the properties of the WMI instance classes involved in the query,
and then performs the necessary data type conversions.

To optimize performance and avoid repeatedly fetching these schema definitions
for every row and every request, an LRU cache is utilized. This cache stores
the schema definition for each unique WMI class encountered. For queries involving
superclasses, such as `CIM_LogicalDevice`, the cache will populate with individual entries
for each specific derived (leaf of the class hierarchy) class whose instances are returned by the query (e.g., `Win32_DiskDrive`, `Win32_NetworkAdapter`, etc.).

::::{info}
**CIM Object Type Support:**
The handling of properties with the `CIM_Object` type (embedded objects) is not supported.
::::

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
    max_rows_per_query: 100
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

**`wmi.include_query_class`**
:   If set to `true` the metricset include the queried class.
Useful if superclasses are queried. The default value is `false`.

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

**`wmi.max_rows_per_query`**
:   A safeguard option to limit the number of rows returned by a single WMI query.
This helps prevent the production of an unexpectedly large amount of data.
The default value is `0`, which is a special value indicating that all fetched
results should be returned without a row limit.

**`wmi.schema_cache_size`**
:   The maximum number of WMI class definitions that can be cached per single query.
Every query keeps its own separate cache.  This cache helps improve performance when dealing with queries that involve inheritance hierarchies. Read more in [WMI Type Support](#wmi-type-support).
For example, if a superclass is queried, the cache
might store all its derived (leaf) classes to optimize subsequent operations.
The default value is `200`.

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

#### Example

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

- Test your WMI queries in isolation using the [`Get-CimInstance`](https://learn.microsoft.com/en-us/powershell/module/cimcmdlets/get-ciminstance) PowerShell cmdlet or [the WMI Explorer tool](https://github.com/vinaypamnani/wmie2).

- Ensure that `wmi.warning_threshold` is **less than or equal to** the module's `period`. This configuration prevents Metricbeat from attempting to start multiple concurrent executions of the same query if a previous one is running slowly.

- Set a `max_rows_per_query` to safeguard against queries with a high number of results

- When possible, try querying **concrete (leaf) classes** or classes closer to the leaves of the WMI inheritance hierarchy. Querying abstract superclasses may require fetching and caching the schema definitions for numerous derived classes, which can lead to increased memory usage and potential cache misses.

- Set up alerts in Metricbeat for documents with the `error.message` field set.

- [Advanced] Configure collection of **WMI-Activity Operational Logs** (found in Event Viewer under `Applications and Services Logs/Microsoft/Windows/WMI-Activity/Operational`). These logs can be invaluable for correlating issues with Metricbeat WMI warnings or documents containing `error.message`.

## Compatibility

This module has been tested on the following platform:

- Operating System: Microsoft Windows Server 2019 Datacenter

- Architecture: x64

Other Windows versions and architectures may also work but have not been
explicitly tested.


