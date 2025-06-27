:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/nginx/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The `nginx` module parses access and error logs created by the [Nginx](http://nginx.org/) HTTP server.

`ingress_controller` fileset parses access logs created by [ingress-nginx](https://github.com/kubernetes/ingress-nginx) controller. Log patterns could be found on the controllers' [docs](https://github.com/kubernetes/ingress-nginx/blob/nginx-0.28.0/docs/user-guide/nginx-configuration/log-format.md).

When you run the module, it performs a few tasks under the hood:

* Sets the default paths to the log files (but don’t worry, you can override the defaults)
* Makes sure each multiline log event gets sent as a single event
* Uses an {{es}} ingest pipeline to parse and process the log lines, shaping the data into a structure suitable for visualizing in Kibana
* Deploys dashboards for visualizing the log data

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Compatibility [_compatibility_26]

The Nginx module was tested with logs from version 1.10.

On Windows, the module was tested with Nginx installed from the Chocolatey repository.

`ingress_controller` fileset was tested with version v0.28.0 and v0.34.1 of `nginx-ingress-controller`.


## Configure the module [configuring-nginx-module]

You can further refine the behavior of the `nginx` module by specifying [variable settings](#nginx-settings) in the `modules.d/nginx.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**

The following example shows how to set paths in the `modules.d/nginx.yml` file to override the default paths for access logs and error logs:

```yaml
- module: nginx
  access:
    enabled: true
    var.paths: ["/path/to/log/nginx/access.log*"]
  error:
    enabled: true
    var.paths: ["/path/to/log/nginx/error.log*"]
```

To specify the same settings at the command line, you use:

```sh
-M "nginx.access.var.paths=[/path/to/log/nginx/access.log*]" -M "nginx.error.var.paths=[/path/to/log/nginx/error.log*]"
```

The following example shows how to configure `ingress_controller` fileset which can be used in Kubernetes environments to parse ingress-nginx logs:

```yaml
- module: nginx
  ingress_controller:
    enabled: true
    var.paths: ["/path/to/log/nginx/ingress.log"]
```


### Variable settings [nginx-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `nginx` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `nginx.access.var.paths` instead of `access.var.paths`.
::::



### `access` log fileset settings [_access_log_fileset_settings_3]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.


### `error` log fileset [_error_log_fileset]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.


### `ingress_controller` log fileset [_ingress_controller_log_fileset]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.


### Time zone support [_time_zone_support_11]

This module parses logs that don’t contain time zone information. For these logs, Filebeat reads the local time zone and uses it when parsing to convert the timestamp to UTC. The time zone to be used for parsing is included in the event in the `event.timezone` field.

To disable this conversion, the `event.timezone` field can be removed with the `drop_fields` processor.

If logs are originated from systems or applications with a different time zone to the local one, the `event.timezone` field can be overwritten with the original time zone using the `add_fields` processor.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


## Example dashboard [_example_dashboard_16]

This module comes with sample dashboards. For example:

% TO DO: Use `:class: screenshot`
![kibana nginx](images/kibana-nginx.png)
