:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/elasticsearch/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This is the elasticsearch module.

When you run the module, it performs a few tasks under the hood:

* Sets the default paths to the log files (but don’t worry, you can override the defaults)
* Makes sure each multiline log event gets sent as a single event
* Uses an {{es}} ingest pipeline to parse and process the log lines, shaping the data into a structure suitable for visualizing in Kibana

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Compatibility [_compatibility_10]

The Elasticsearch module is compatible with Elasticsearch 6.2 and newer.


## Configure the module [configuring-elasticsearch-module]

You can further refine the behavior of the `elasticsearch` module by specifying [variable settings](#elasticsearch-settings) in the `modules.d/elasticsearch.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**


### Variable settings [elasticsearch-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `elasticsearch` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `elasticsearch.server.var.paths` instead of `server.var.paths`.
::::



### `server` log fileset settings [_server_log_fileset_settings]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.

    Example config:

    ```yaml
      server:
        enabled: true
        var.paths:
          - /var/log/elasticsearch/*.log          # Plain text logs
          - /var/log/elasticsearch/*_server.json  # JSON logs
    ```

    ::::{note}
    If you’re running against Elasticsearch >= 7.0.0, configure the `var.paths` setting to point to JSON logs. Otherwise, configure it to point to plain text logs.
    ::::



### `gc` log fileset settings [_gc_log_fileset_settings]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.

    Example config:

    ```yaml
      gc:
        var.paths:
          - /var/log/elasticsearch/gc.log.[0-9]*
          - /var/log/elasticsearch/gc.log
    ```



### `audit` log fileset settings [_audit_log_fileset_settings_2]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.

    Example config:

    ```yaml
      audit:
        var.paths:
          - /var/log/elasticsearch/*_access.log  # Plain text logs
          - /var/log/elasticsearch/*_audit.json  # JSON logs
    ```

    ::::{note}
    If you’re running against Elasticsearch >= 7.0.0, configure the `var.paths` setting to point to JSON logs. Otherwise, configure it to point to plain text logs.
    ::::



### `slowlog` log fileset settings [_slowlog_log_fileset_settings]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.

    Example config:

    ```yaml
      slowlog:
        var.paths:
          - /var/log/elasticsearch/*_index_search_slowlog.log     # Plain text logs
          - /var/log/elasticsearch/*_index_indexing_slowlog.log   # Plain text logs
          - /var/log/elasticsearch/*_index_search_slowlog.json    # JSON logs
          - /var/log/elasticsearch/*_index_indexing_slowlog.json  # JSON logs
    ```

    ::::{note}
    If you’re running against Elasticsearch >= 7.0.0, configure the `var.paths` setting to point to JSON logs. Otherwise, configure it to point to plain text logs.
    ::::



### `deprecation` log fileset settings [_deprecation_log_fileset_settings]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.

    Example config:

    ```yaml
      deprecation:
        var.paths:
          - /var/log/elasticsearch/*_deprecation.log   # Plain text logs
          - /var/log/elasticsearch/*_deprecation.json  # JSON logs
    ```

    ::::{note}
    If you’re running against Elasticsearch >= 7.0.0, configure the `var.paths` setting to point to JSON logs. Otherwise, configure it to point to plain text logs.
    ::::



### Time zone support [_time_zone_support_4]

This module parses logs that don’t contain time zone information. For these logs, Filebeat reads the local time zone and uses it when parsing to convert the timestamp to UTC. The time zone to be used for parsing is included in the event in the `event.timezone` field.

To disable this conversion, the `event.timezone` field can be removed with the `drop_fields` processor.

If logs are originated from systems or applications with a different time zone to the local one, the `event.timezone` field can be overwritten with the original time zone using the `add_fields` processor.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.
