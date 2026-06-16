:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/auditd/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The `auditd` module collects and parses logs from the audit daemon (`auditd`).

::::{note}
Although Filebeat is able to parse logs by using the `auditd` module, [{{auditbeat}}](/reference/auditbeat/auditbeat-module-auditd.md) offers more advanced features for monitoring audit logs.
::::


When you run the module, it performs a few tasks under the hood:

* Sets the default paths to the log files (but don’t worry, you can override the defaults)
* Makes sure each multiline log event gets sent as a single event
* Uses an {{es}} ingest pipeline to parse and process the log lines, shaping the data into a structure suitable for visualizing in Kibana
* Deploys dashboards for visualizing the log data

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Compatibility [_compatibility_6]

The `auditd` module was tested with logs from `auditd` on OSes like CentOS 6 and CentOS 7.

This module is not available for Windows.


## Configure the module [configuring-auditd-module]

You can further refine the behavior of the `auditd` module by specifying [variable settings](#auditd-settings) in the `modules.d/auditd.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**

The following example shows how to set paths in the `modules.d/auditd.yml` file to override the default paths for logs:

```yaml
- module: auditd
  log:
    enabled: true
    var.paths: ["/path/to/log/audit/audit.log*"]
```

To specify the same settings at the command line, you use:

```sh
-M "auditd.log.var.paths=[/path/to/log/audit/audit.log*]"
```


### Variable settings [auditd-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `auditd` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `auditd.log.var.paths` instead of `log.var.paths`.
::::



### `log` fileset settings [_log_fileset_settings]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.


## Example dashboard [_example_dashboard_2]

This module comes with a sample dashboard showing an overview of the audit log data. You can build more specific dashboards that are tailored to the audit rules that you use on your systems.

% TO DO: Use `:class: screenshot`
![kibana audit auditd](images/kibana-audit-auditd.png)
