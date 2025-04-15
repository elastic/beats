---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-module-mysqlenterprise.html
---

# MySQL Enterprise module [filebeat-module-mysqlenterprise]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This is a module for different types of MySQL logs. Currently focusing on data from the MySQL Enterprise Audit Plugin in JSON format.

To configure the the Enterprise Audit Plugin to output in JSON format please follow the directions in the [MySQL Documentation.](https://dev.mysql.com/doc/refman/8.0/en/audit-log-file-formats.md)

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Compatibility [_compatibility_24]

This module has been tested against MySQL Enterprise 5.7.x and 8.0.x


## Configure the module [configuring-mysqlenterprise-module]

You can further refine the behavior of the `mysqlenterprise` module by specifying [variable settings](#mysqlenterprise-settings) in the `modules.d/mysqlenterprise.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**


### Variable settings [mysqlenterprise-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `mysqlenterprise` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `mysqlenterprise.audit.var.paths` instead of `audit.var.paths`.
::::



### `audit` fileset settings [_audit_fileset_settings_4]

Example config:

```yaml
- module: mysqlenterprise
  audit:
    var.input: file
    var.paths: /home/user/mysqlauditlogs/audit.*.log
```

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.

**`var.tags`**
:   A list of tags to include in events. Including `forwarded` indicates that the events did not originate on this host and causes `host.name` to not be added to events. Defaults to `[mysqlenterprise-audit]`.


### MySQL Enterprise ECS Fields [_mysql_enterprise_ecs_fields]

MySQL Enterprise Audit fields are mapped to ECS in the following way:

| MySQL Enterprise Fields | ECS Fields |  |
| --- | --- | --- |
| account.user | server.user.name |  |
| account.host | client.domain |  |
| login.os | client.user.name |  |
| login.ip | client.ip |  |
| startup_data.os_version | host.os.full |  |
| startup_data.args | process.args |  |
| connection_attributes._pid | process.pid |  |
| timestamp | @timestamp |  |


## Fields [_fields_34]

For a description of each field in the module, see the [exported fields](/reference/filebeat/exported-fields-mysqlenterprise.md) section.
