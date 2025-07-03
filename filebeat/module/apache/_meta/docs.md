:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/apache/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The `apache` module parses access and error logs created by the [Apache HTTP](https://httpd.apache.org/) server.

When you run the module, it performs a few tasks under the hood:

* Sets the default paths to the log files (but don’t worry, you can override the defaults)
* Makes sure each multiline log event gets sent as a single event
* Uses an {{es}} ingest pipeline to parse and process the log lines, shaping the data into a structure suitable for visualizing in Kibana
* Deploys dashboards for visualizing the log data

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Compatibility [_compatibility_5]

The `apache` module was tested with logs from versions 2.2.22 and 2.4.23.

On Windows, the module was tested with Apache HTTP Server installed from the Chocolatey repository.


## Configure the module [configuring-apache-module]

You can further refine the behavior of the `apache` module by specifying [variable settings](#apache-settings) in the `modules.d/apache.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**

The following example shows how to set paths in the `modules.d/apache.yml` file to override the default paths for Apache HTTP Server access and error logs:

```yaml
- module: apache
  access:
    enabled: true
    var.paths: ["/path/to/log/apache/access.log*"]
  error:
    enabled: true
    var.paths: ["/path/to/log/apache/error.log*"]
```

To specify the same settings at the command line, you use:

```sh
-M "apache.access.var.paths=[/path/to/apache/access.log*]" -M "apache.error.var.paths=[/path/to/log/apache/error.log*]"
```


### Variable settings [apache-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `apache` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `apache.access.var.paths` instead of `access.var.paths`.
::::



### `access` log fileset settings [_access_log_fileset_settings]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.


### `error` log fileset settings [_error_log_fileset_settings]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.


### Time zone support [_time_zone_support_2]

This module parses logs that don’t contain time zone information. For these logs, Filebeat reads the local time zone and uses it when parsing to convert the timestamp to UTC. The time zone to be used for parsing is included in the event in the `event.timezone` field.

To disable this conversion, the `event.timezone` field can be removed with the `drop_fields` processor.

If logs are originated from systems or applications with a different time zone to the local one, the `event.timezone` field can be overwritten with the original time zone using the `add_fields` processor.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


## Virtual Host [_virtual_host]

See customlog documentation  [https://httpd.apache.org/docs/2.4/en/mod/mod_log_config.html](https://httpd.apache.org/docs/2.4/en/mod/mod_log_config.html) Add %v config in httpd.conf in log section

```sh
    # Replace
    LogFormat "%h %l %u %t \"%r\" %>s %b \"%{Referer}i\" \"%{User-Agent}i\"" combined
    # By
    LogFormat "%v %h %l %u %t \"%r\" %>s %b \"%{Referer}i\" \"%{User-Agent}i\"" combined
```


## Example dashboard [_example_dashboard]

This module comes with a sample dashboard. For example:

% TO DO: Use `:class: screenshot`
![kibana apache](images/kibana-apache.png)
