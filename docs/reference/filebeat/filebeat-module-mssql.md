---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-module-mssql.html
---

# MSSQL module [filebeat-module-mssql]

The `mssql` module parses error logs created by MSSQL.

When you run the module, it performs a few tasks under the hood:

* Sets the default paths to the log files (but don’t worry, you can override the defaults)
* Makes sure each multiline log event gets sent as a single event
* Uses an {{es}} ingest pipeline to parse and process the log lines, shaping the data into a structure suitable for visualizing in Kibana

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Configure the module [configuring-mssql-module]

You can further refine the behavior of the `mssql` module by specifying [variable settings](#mssql-settings) in the `modules.d/mssql.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**

The following example shows how to set paths in the `modules.d/mssql.yml` file to override the default paths for MSSQL logs:

```yaml
- module: mssql
  log:
    enabled: true
    var.paths: ['C:\Program Files\Microsoft SQL Server\MSSQL.150\MSSQL\LOG\ERRORLOG*']
```

To specify the same settings at the command line, you use:

```sh
-M "mssql.log.var.paths=['C:\Program Files\Microsoft SQL Server\MSSQL.150\MSSQL\LOG\ERRORLOG*']"
```


### Variable settings [mssql-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `mssql` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `mssql.log.var.paths` instead of `log.var.paths`.
::::



### `log` fileset settings [_log_fileset_settings_8]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.


### Time zone support [_time_zone_support_10]

This module parses logs that don’t contain time zone information. For these logs, Filebeat reads the local time zone and uses it when parsing to convert the timestamp to UTC. The time zone to be used for parsing is included in the event in the `event.timezone` field.

To disable this conversion, the `event.timezone` field can be removed with the `drop_fields` processor.

If logs are originated from systems or applications with a different time zone to the local one, the `event.timezone` field can be overwritten with the original time zone using the `add_fields` processor.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


## Fields [_fields_32]

For a description of each field in the module, see the [exported fields](/reference/filebeat/exported-fields-mssql.md) section.
