:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/logstash/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The `logstash` modules parse logstash regular logs and the slow log, it will support the plain text format and the JSON format.

When you run the module, it performs a few tasks under the hood:

* Sets the default paths to the log files (but don’t worry, you can override the defaults)
* Makes sure each multiline log event gets sent as a single event
* Uses an {{es}} ingest pipeline to parse and process the log lines, shaping the data into a structure suitable for visualizing in Kibana
* Deploys dashboards for visualizing the log data

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::


The `logstash` module has two filesets:

* The `log` fileset collects and parses the logs that Logstash writes to disk.
* The `slowlog` fileset parses the logstash slowlog.

For the `slowlog` fileset, make sure to configure the [Logstash slowlog option](logstash://reference/logging.md#_slowlog).


## Compatibility [_compatibility_21]

The Logstash `log` fileset was tested with logs from Logstash 5.6 and 6.0.

The Logstash `slowlog` fileset was tested with logs from Logstash 5.6 and 6.0


## Configure the module [configuring-logstash-module]

You can further refine the behavior of the `logstash` module by specifying [variable settings](#logstash-settings) in the `modules.d/logstash.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**

The following example shows how to set paths in the `modules.d/logstash.yml` file to override the default paths for Logstash logs.

```yaml
- module: logstash
  log:
    enabled: true
    var.paths: ["/path/to/log/logstash.log*"]
  slowlog:
    enabled: true
    var.paths: ["/path/to/log/logstash-slowlog.log*"]
```

To specify the same settings at the command line, you use:

```sh
-M "logstash.log.var.paths=[/path/to/log/logstash/logstash-server.log*]" -M "logstash.slowlog.var.paths=[/path/to/log/logstash/logstash-slowlog.log*]"
```


### Variable settings [logstash-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `logstash` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `logstash.log.var.paths` instead of `log.var.paths`.
::::



### `log` fileset settings [_log_fileset_settings_7]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.


### `slowlog` fileset settings [_slowlog_fileset_settings]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.


### Time zone support [_time_zone_support_9]

This module parses logs that don’t contain time zone information. For these logs, Filebeat reads the local time zone and uses it when parsing to convert the timestamp to UTC. The time zone to be used for parsing is included in the event in the `event.timezone` field.

To disable this conversion, the `event.timezone` field can be removed with the `drop_fields` processor.

If logs are originated from systems or applications with a different time zone to the local one, the `event.timezone` field can be overwritten with the original time zone using the `add_fields` processor.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


## Example dashboards [_example_dashboards]

This module comes with two sample dashboards.

% TO DO: Use `:class: screenshot`
![kibana logstash log](images/kibana-logstash-log.png)

% TO DO: Use `:class: screenshot`
![kibana logstash slowlog](images/kibana-logstash-slowlog.png)


## Known issues [_known_issues]

When using the `log` fileset to parse plaintext logs, if a multiline plaintext log contains an embedded JSON object such that the JSON object starts on a new line, the fileset may not parse the multiline plaintext log event correctly.
