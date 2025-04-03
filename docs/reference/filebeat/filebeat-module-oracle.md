---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-module-oracle.html
---

# Oracle module [filebeat-module-oracle]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/oracle/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This is a module for ingesting Audit Trail logs from Oracle Databases.

The module expects an *.aud audit file that is generated from Oracle Databases by default. If this has been disabled then please see the [Oracle Database Audit Trail Documentation](https://docs.oracle.com/en/database/oracle/oracle-database/19/dbseg/introduction-to-auditing.html#GUID-8D96829C-9151-4FA4-BED9-831D088F12FF).

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Compatibility [_compatibility_27]

This module has been tested with Oracle Database 19c, and should work for 18c as well though it has not been tested.


## Configure the module [configuring-oracle-module]

You can further refine the behavior of the `oracle` module by specifying [variable settings](#oracle-settings) in the `modules.d/oracle.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**


### Variable settings [oracle-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `oracle` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `oracle.database_audit.var.paths` instead of `database_audit.var.paths`.
::::



### `database_audit` fileset settings [_database_audit_fileset_settings]

Example config:

```yaml
- module: oracle
  database_audit:
    var.input: file
    var.paths: /home/user/oracleauditlogs/*/*.aud
```

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.

**`var.tags`**
:   A list of tags to include in events. Including `forwarded` indicates that the events did not originate on this host and causes `host.name` to not be added to events. Defaults to `[oracle-database-audit]`.


### Oracle Database fields [_oracle_database_fields]

Oracle Database fields are mapped to the current ECS Fields:

| Oracle Fields | ECS Fields |  |
| --- | --- | --- |
| privilege | host.user.roles |  |
| client_user | client.user.name |  |
| userhost | client.ip/domain |  |
| database_user | server.user.name |  |


## Fields [_fields_40]

For a description of each field in the module, see the [exported fields](/reference/filebeat/exported-fields-oracle.md) section.
