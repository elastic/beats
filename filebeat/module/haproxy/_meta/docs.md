:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/haproxy/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The `haproxy` module collects and parses logs from a (`haproxy`) process.

When you run the module, it performs a few tasks under the hood:

* Sets the default paths to the log files (but don’t worry, you can override the defaults)
* Makes sure each multiline log event gets sent as a single event
* Uses an {{es}} ingest pipeline to parse and process the log lines, shaping the data into a structure suitable for visualizing in Kibana
* Deploys dashboards for visualizing the log data

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Compatibility [_compatibility_14]

The `haproxy` module was tested with logs from `haproxy` running on AWS Linux as a gateway to a cluster of microservices.

The module was also tested with HAProxy 1.8, 1.9 and 2.0 running on a Debian.

This module is not available for Windows.


## Configure the module [configuring-haproxy-module]

You can further refine the behavior of the `haproxy` module by specifying [variable settings](#haproxy-settings) in the `modules.d/haproxy.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**

The module is by default configured to run via syslog on port 9001. However it can also be configured to read from a file path. See the following example.

```yaml
- module: haproxy
  log:
    enabled: true
    var.paths: ["/var/log/haproxy.log"]
    var.input: "file"
```


### Variable settings [haproxy-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `haproxy` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `haproxy.log.var.paths` instead of `log.var.paths`.
::::



### `log` fileset settings [_log_fileset_settings_4]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.


### Time zone support [_time_zone_support_6]

This module parses logs that don’t contain time zone information. For these logs, Filebeat reads the local time zone and uses it when parsing to convert the timestamp to UTC. The time zone to be used for parsing is included in the event in the `event.timezone` field.

To disable this conversion, the `event.timezone` field can be removed with the `drop_fields` processor.

If logs are originated from systems or applications with a different time zone to the local one, the `event.timezone` field can be overwritten with the original time zone using the `add_fields` processor.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


## Example dashboard [_example_dashboard_7]

This module comes with a sample dashboard showing geolocation, distribution of requests between backends and frontends, and status codes over time. For example:

% TO DO: Use `:class: screenshot`
![kibana haproxy overview](images/kibana-haproxy-overview.png)
