:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/santa/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The `santa` module collects and parses logs from [Google Santa](https://github.com/google/santa), a security tool for macOS that monitors process executions and can blacklist/whitelist binaries.

When you run the module, it performs a few tasks under the hood:

* Sets the default paths to the log files (but don’t worry, you can override the defaults)
* Makes sure each multiline log event gets sent as a single event
* Uses an {{es}} ingest pipeline to parse and process the log lines, shaping the data into a structure suitable for visualizing in Kibana
* Deploys dashboards for visualizing the log data

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Compatibility [_compatibility_34]

The `santa` module was tested with logs from Santa 0.9.14.

This module is available for MacOS only.


## Configure the module [configuring-santa-module]

You can further refine the behavior of the `santa` module by specifying [variable settings](#santa-settings) in the `modules.d/santa.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**

The module is by default configured to read logs from `/var/log/santa.log`.

```yaml
- module: santa
  log:
    enabled: true
    var.paths: ["/var/log/santa.log"]
    var.input: "file"
```


### Variable settings [santa-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `santa` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `santa.log.var.paths` instead of `log.var.paths`.
::::



### `log` fileset settings [_log_fileset_settings_13]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.


## Example dashboard [_example_dashboard_23]

This module comes with a sample dashboard showing and overview of the processes that are executing.

% TO DO: Use `:class: screenshot`
![kibana santa log overview](images/kibana-santa-log-overview.png)
