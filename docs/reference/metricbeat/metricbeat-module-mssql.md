---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-mssql.html
---

# MSSQL module [metricbeat-module-mssql]

This is the [Microsoft SQL 2017](https://www.microsoft.com/en-us/sql-server/sql-server-2017) Metricbeat module. It is still under active development to add new Metricsets and introduce enhancements.


## Compatibility [_compatibility_35]

The module is being tested with [2017 GA](https://hub.docker.com/r/microsoft/mssql-server-linux/) version under Linux


## Permission/Access required for tables [_permissionaccess_required_for_tables]

1.`transaction_log` :

* sys.databases
* sys.dm_db_log_space_usage
* sys.dm_db_log_stats(DB_ID)

2.`performance` :

* sys.dm_os_performance_counters

If you browse MSDN for above tables, you will find "Permissions" section which defines the permission needed, e.g [Permissions](https://docs.microsoft.com/en-us/sql/relational-databases/system-dynamic-management-views/sys-dm-db-log-space-usage-transact-sql?view=sql-server-ver15)


## Metricsets [_metricsets_50]

The following Metricsets are already included:


### `transaction_log` [_transaction_log]

`transaction_log` Metricset fetches information about the operation and transaction log of each MSSQL database in the monitored instance. All data is extracted from the [Database Dynamic Management Views](https://docs.microsoft.com/en-us/sql/relational-databases/system-dynamic-management-views/database-related-dynamic-management-views-transact-sql?view=sql-server-2017)


### `performance` [_performance]

`performance` Metricset fetches information from what’s commonly known as [Performance Counters](https://docs.microsoft.com/en-us/sql/relational-databases/system-dynamic-management-views/sys-dm-os-performance-counters-transact-sql?view=sql-server-2017) in MSSQL.


## Module-specific configuration notes [_module_specific_configuration_notes_12]

When configuring the `hosts` option, you can specify native user credentials as part of the host string with the following format:

```
hosts: ["sqlserver://sa@localhost"]]
```

To use Active Directory domain credentials, you can separately specify the username and password using the respective configuration options to allow the domain to be included in the username:

```
metricbeat.modules:
- module: mssql
  metricsets:
    - "transaction_log"
    - "performance"
  hosts: ["sqlserver://localhost"]
  username: domain\username
  password: verysecurepassword
  period: 10
```

Store sensitive values like passwords in the [secrets keystore](/reference/metricbeat/keystore.md).


## Example configuration [_example_configuration_44]

The MSSQL module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: mssql
  metricsets:
    - "transaction_log"
    - "performance"
  hosts: ["sqlserver://localhost"]
  username: domain\username
  password: verysecurepassword
  period: 10s
```


## Metricsets [_metricsets_51]

The following metricsets are available:

* [performance](/reference/metricbeat/metricbeat-metricset-mssql-performance.md)
* [transaction_log](/reference/metricbeat/metricbeat-metricset-mssql-transaction_log.md)



