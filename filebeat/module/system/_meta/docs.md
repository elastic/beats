:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/system/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The `system` module collects and parses logs created by the system logging service of common Unix/Linux based distributions.

When you run the module, it performs a few tasks under the hood:

* Sets the default paths to the log files (but don’t worry, you can override the defaults)
* Makes sure each multiline log event gets sent as a single event
* Uses an {{es}} ingest pipeline to parse and process the log lines, shaping the data into a structure suitable for visualizing in Kibana
* Deploys dashboards for visualizing the log data

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Compatibility [_compatibility_37]

This module was tested with logs from OSes like Ubuntu 12.04, Centos 7, and macOS Sierra.

This module is not available for Windows.


## Configure the module [configuring-system-module]

You can further refine the behavior of the `system` module by specifying [variable settings](#system-settings) in the `modules.d/system.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**

The following example shows how to set paths in the `modules.d/system.yml` file to override the default paths for the syslog and authorization logs:

```yaml
- module: system
  syslog:
    enabled: true
    var.paths: ["/path/to/log/syslog*"]
  auth:
    enabled: true
    var.paths: ["/path/to/log/auth.log*"]
```

To specify the same settings at the command line, you use:

```sh
-M "system.syslog.var.paths=[/path/to/log/syslog*]" -M "system.auth.var.paths=[/path/to/log/auth.log*]"
```


### Variable settings [system-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `system` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `system.syslog.var.paths` instead of `syslog.var.paths`.
::::



### `syslog` fileset settings [_syslog_fileset_settings]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.

**`var.use_journald`**
:   A boolean that when set to `true` will read logs from Journald. When Journald is used all events contain the tag `journald`.


### `auth` fileset settings [_auth_fileset_settings]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.

**`var.use_journald`**
:   A boolean that when set to `true` will read logs from Journald. When Journald is used all events contain the tag `journald`.

**`var.tags`**
:   A list of tags to include in events. Including `forwarded` indicates that the events did not originate on this host and causes `host.name` to not be added to events. Include `preserve_orginal_event` causes the pipeline to retain the raw log in `event.original`. Defaults to `[]`.


### Time zone support [_time_zone_support_14]

This module parses logs that don’t contain time zone information. For these logs, Filebeat reads the local time zone and uses it when parsing to convert the timestamp to UTC. The time zone to be used for parsing is included in the event in the `event.timezone` field.

To disable this conversion, the `event.timezone` field can be removed with the `drop_fields` processor.

If logs are originated from systems or applications with a different time zone to the local one, the `event.timezone` field can be overwritten with the original time zone using the `add_fields` processor.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


## Example dashboards [_example_dashboards_3]

This module comes with sample dashboards. For example:

% TO DO: Use `:class: screenshot`
![kibana system](images/kibana-system.png)
