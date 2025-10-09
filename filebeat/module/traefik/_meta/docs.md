:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/traefik/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


The `traefik` module parses access logs created by [Træfik](https://traefik.io/).

When you run the module, it performs a few tasks under the hood:

* Sets the default paths to the log files (but don’t worry, you can override the defaults)
* Makes sure each multiline log event gets sent as a single event
* Uses an {{es}} ingest pipeline to parse and process the log lines, shaping the data into a structure suitable for visualizing in Kibana
* Deploys dashboards for visualizing the log data

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Configure the module [configuring-traefik-module]

You can further refine the behavior of the `traefik` module by specifying [variable settings](#traefik-settings) in the `modules.d/traefik.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**

The following example shows how to set paths in the `modules.d/traefik.yml` file to override the default paths for Træfik logs:

```yaml
- module: traefik
  access:
    enabled: true
    var.paths: ["/usr/local/traefik/access.log*"]
```

To specify the same settings at the command line, you use:

```sh
-M "traefik.access.var.paths=[/path/to/traefik/access.log*]"
```


### Variable settings [traefik-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `traefik` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `traefik.access.var.paths` instead of `access.var.paths`.
::::



### `access` log fileset settings [_access_log_fileset_settings_4]

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.


## Example dashboards [_example_dashboards_4]

This module comes with sample dashboards. For example:

% TO DO: Use `:class: screenshot`
![kibana traefik](images/kibana-traefik.png)
