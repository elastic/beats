:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/iis/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The `iis` module parses access and error logs created by the Internet Information Services (IIS) HTTP server.

::::{important}
The `iis` module currently supports only the default W3C log format.

::::


When you run the module, it performs a few tasks under the hood:

* Sets the default paths to the log files (but don’t worry, you can override the defaults)
* Makes sure each multiline log event gets sent as a single event
* Uses an {{es}} ingest pipeline to parse and process the log lines, shaping the data into a structure suitable for visualizing in Kibana
* Deploys dashboards for visualizing the log data

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Compatibility [_compatibility_17]

The IIS module was tested with logs from version 7.5 and version 10.


## Configure the module [configuring-iis-module]

You can further refine the behavior of the `iis` module by specifying [variable settings](#iis-settings) in the `modules.d/iis.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**

The following example shows how to set paths in the `modules.d/iis.yml` file to override the default paths for IIS access logs and error logs:

```yaml
- module: iis
  access:
    enabled: true
    var.paths: ["C:/inetpub/logs/LogFiles/*/*.log"]
  error:
    enabled: true
    var.paths: ["C:/Windows/System32/LogFiles/HTTPERR/*.log"]
```

To specify the same settings at the command line, you use:

```sh
-M "iis.access.var.paths=[C:/inetpub/logs/LogFiles/*/*.log]" -M "iis.error.var.paths=[C:/Windows/System32/LogFiles/HTTPERR/*.log]"
```


### Variable settings [iis-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `iis` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `iis.access.var.paths` instead of `access.var.paths`.
::::



### `access` log fileset settings [_access_log_fileset_settings_2]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.


### `error` log fileset settings [_error_log_fileset_settings_2]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.


## Example dashboard [_example_dashboard_10]

This module comes with a sample dashboard. For example:

% TO DO: Use `:class: screenshot`
![kibana iis](images/kibana-iis.png)
