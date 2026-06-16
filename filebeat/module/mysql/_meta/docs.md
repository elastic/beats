:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/mysql/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The `mysql` module collects and parses the slow logs and error logs created by [MySQL](https://www.mysql.com/).

When you run the module, it performs a few tasks under the hood:

* Sets the default paths to the log files (but don’t worry, you can override the defaults)
* Makes sure each multiline log event gets sent as a single event
* Uses an {{es}} ingest pipeline to parse and process the log lines, shaping the data into a structure suitable for visualizing in Kibana
* Deploys dashboards for visualizing the log data

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Compatibility [_compatibility_23]

The  `mysql` module was tested with logs from MySQL 5.5, 5.7 and 8.0, MariaDB 10.1, 10.2 and 10.3, and Percona 5.7 and 8.0.

On Windows, the module was tested with MySQL installed from the Chocolatey repository.


## Configure the module [configuring-mysql-module]

You can further refine the behavior of the `mysql` module by specifying [variable settings](#mysql-settings) in the `modules.d/mysql.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**

The following example shows how to set paths in the `modules.d/mysql.yml` file to override the default paths for slow logs and error logs:

```yaml
- module: mysql
  error:
    enabled: true
    var.paths: ["/path/to/log/mysql/error.log*"]
  slowlog:
    enabled: true
    var.paths: ["/path/to/log/mysql/mysql-slow.log*"]
```

To specify the same settings at the command line, you use:

```sh
-M "mysql.error.var.paths=[/path/to/log/mysql/error.log*]" -M "mysql.slowlog.var.paths=[/path/to/log/mysql/mysql-slow.log*]"
```


### Variable settings [mysql-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `mysql` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `mysql.error.var.paths` instead of `error.var.paths`.
::::



### `error` log fileset settings [_error_log_fileset_settings_3]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.


### `slowlog` fileset settings [_slowlog_fileset_settings_2]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.


## Example dashboard [_example_dashboard_15]

This module comes with a sample dashboard. For example:

% TO DO: Use `:class: screenshot`
![kibana mysql](images/kibana-mysql.png)
