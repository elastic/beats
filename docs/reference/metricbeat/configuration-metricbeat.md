---
navigation_title: "Modules"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/configuration-metricbeat.html
---

# Configure modules [configuration-metricbeat]


You can configure modules in the `modules.d` directory (recommended), or in the Metricbeat configuration file. Settings for enabled modules in the `modules.d` directory take precedence over module settings in the Metricbeat configuration file.

Before running Metricbeat with modules enabled, make sure you also set up the environment to use {{kib}} dashboards. See [Quick start: installation and configuration](/reference/metricbeat/metricbeat-installation-configuration.md) for more information.

::::{note}
On systems with POSIX file permissions, all Beats configuration files are subject to ownership and file permission checks. For more information, see [Config File Ownership and Permissions](/reference/libbeat/config-file-permissions.md).
::::



### Configure modules in the `modules.d` directory [configure-modules-d-configs]

The `modules.d` directory contains default configurations for all the modules available in Metricbeat. To enable or disable specific module configurations under `modules.d`, run the [`modules enable` or `modules disable`](/reference/metricbeat/command-line-options.md#modules-command) command. For example:

:::::::{tab-set}

::::::{tab-item} DEB
```sh
metricbeat modules enable apache mysql
```
::::::

::::::{tab-item} RPM
```sh
metricbeat modules enable apache mysql
```
::::::

::::::{tab-item} MacOS
```sh
./metricbeat modules enable apache mysql
```
::::::

::::::{tab-item} Linux
```sh
./metricbeat modules enable apache mysql
```
::::::

::::::{tab-item} Windows
```sh
PS > .\metricbeat.exe modules enable apache mysql
```
::::::

:::::::
Then when you run Metricbeat, it loads the corresponding module configurations specified in the `modules.d` directory (for example, `modules.d/apache.yml` and `modules.d/mysql.yml`).

To see a list of enabled and disabled modules, run:

:::::::{tab-set}

::::::{tab-item} DEB
```sh
metricbeat modules list
```
::::::

::::::{tab-item} RPM
```sh
metricbeat modules list
```
::::::

::::::{tab-item} MacOS
```sh
./metricbeat modules list
```
::::::

::::::{tab-item} Linux
```sh
./metricbeat modules list
```
::::::

::::::{tab-item} Windows
```sh
PS > .\metricbeat.exe modules list
```
::::::

:::::::
To change the default module configurations, modify the `.yml` files in the `modules.d` directory.

The following example shows a basic configuration for the Apache module:

```yaml
- module: apache
  metricsets: ["status"]
  hosts: ["http://127.0.0.1/"]
  period: 10s
  fields:
    dc: west
  tags: ["tag"]
  processors:
  ....
```

See [Configuration combinations](#config-combos) for additional configuration examples.


### Configure modules in the `metricbeat.yml` file [configure-modules-config-file]

When possible, you should use the config files in the `modules.d` directory.

However, configuring [modules](/reference/metricbeat/metricbeat-modules.md) directly in the config file is a practical approach if you have upgraded from a previous version of Metricbeat and don’t want to move your module configs to the `modules.d` directory. You can continue to configure modules in the `metricbeat.yml` file, but you won’t be able to use the `modules` command to enable and disable configurations because the command requires the `modules.d` layout.

To enable specific modules and metricsets in the `metricbeat.yml` config file, add entries to the `metricbeat.modules` list. Each entry in the list begins with a dash (-) and is followed by settings for that module.

::::{tip}
Check the `modules.d` directory to verify that the modules you’ve specified in `metricbeat.yml` are disabled (filename ends with `.disabled`). If they aren’t, disable them now by running `metricbeat modules disable <modulename>`.
::::


The following example shows a configuration where the `apache` and `mysql` modules are enabled:

```yaml
metricbeat.modules:

#---------------------------- Apache Status Module ---------------------------
- module: apache
  metricsets: ["status"]
  period: 1s
  hosts: ["http://127.0.0.1/"]

#---------------------------- MySQL Status Module ----------------------------
- module: mysql
  metricsets: ["status"]
  period: 2s
  hosts: ["root@tcp(127.0.0.1:3306)/"]
```

In the following example, the Redis host is crawled for `stats` information every second because this is critical data, but the full list of Apache metricsets is only fetched every 30 seconds because the metrics are less critical.

```yaml
metricbeat.modules:
- module: redis
  metricsets: ["info"]
  hosts: ["host1"]
  period: 1s
- module: apache
  metricsets: ["info"]
  hosts: ["host1"]
  period: 30s
```


## Configuration variants [config-variants]

Every module comes with a default configuration file. Some modules also come with one or more variant configuration files containing common alternative configurations for that module.

When you see the list of enabled and disabled modules, those modules with configuration variants will be shown as `<module_name>-<variant_name>`. You can enable or disable specific configuration variants of a module by specifying `metricbeat modules enable <module_name>-<variant_name>` and `metricbeat modules disable <module_name>-<variant_name>` respectively.


## Configuration combinations [config-combos]

You can specify a module configuration that uses different combinations of metricsets, periods, and hosts.

For a module with multiple metricsets defined, it’s possible to define the module twice and specify a different period to use for each metricset. For the following example, the `set1` metricset will be fetched every 10 seconds, while the `set2` metricset will be fetched every 2 minutes:

```yaml
- module: example
  metricsets: ["set1"]
  hosts: ["host1"]
  period: 10s
- module: example
  metricsets: ["set2"]
  hosts: ["host1"]
  period: 2m
```


### Standard config options [module-config-options]

You can specify the following options for any Metricbeat module. Some modules require additional configuration settings. See the [Modules](/reference/metricbeat/metricbeat-modules.md) section for more information.


#### `module` [_module]

The name of the module to run. For documentation about each module, see the [Modules](/reference/metricbeat/metricbeat-modules.md) section.


#### `metricsets` [_metricsets]

A list of metricsets to execute. Make sure that you only list metricsets that are available in the module. It is not possible to reference metricsets from other modules. For a list of available metricsets, see [Modules](/reference/metricbeat/metricbeat-modules.md).


#### `enabled` [_enabled]

A Boolean value that specifies whether the module is enabled. If you use the default config file, `metricbeat.yml`, the System module is enabled (set to `enabled: true`) by default. If the `enabled` option is missing from the configuration block, the module is enabled by default.


#### `period` [metricset-period]

How often the metricsets are executed. If a system is not reachable, Metricbeat returns an error for each period. This setting is required.


#### `hosts` [_hosts]

A list of hosts to fetch information from. For some metricsets, such as the System module, this setting is optional.


#### `fields` [_fields]

A dictionary of fields that will be sent with the metricset event. This setting is optional.


#### `tags` [_tags]

A list of tags that will be sent with the metricset event. This setting is optional.


#### `processors` [_processors]

A list of processors to apply to the data generated by the metricset.

See [Processors](/reference/metricbeat/filtering-enhancing-data.md) for information about specifying processors in your config.


#### `index` [_index]

If present, this formatted string overrides the index for events from this module (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"metricbeat-myindex-2019.12.13"`.


#### `keep_null` [_keep_null]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `service.name` [_service_name]

A name given by the user to the service the data is collected from. It can be used for example to identify information collected from nodes of different clusters with the same `service.type`.


### Standard HTTP config options [module-http-config-options]

Modules and metricsets that define the host as an HTTP URL can use the standard schemes for HTTP (`http://` and `https://`) and the following schemes to connect to local pipes:

* `http+unix://` to connect to UNIX sockets.
* `http+npipe://` to connect to Windows named pipes.

The following options are available for modules and metricsets that define the host as an HTTP URL:


#### `username` [_username]

The username to use for basic authentication.


#### `password` [_password]

The password to use for basic authentication.


#### `connect_timeout` [_connect_timeout]

Total time limit for an HTTP connection to be completed (Default: 2 seconds).


#### `timeout` [_timeout]

Total time limit for HTTP requests made by the module (Default: 10 seconds).


#### `ssl` [_ssl]

Configuration options for SSL parameters like the certificate authority to use for HTTPS-based connections.

See [SSL](/reference/metricbeat/configuration-ssl.md) for more information.


#### `headers` [_headers]

A list of headers to use with the HTTP request. For example:

```yaml
headers:
  Cookie: abcdef=123456
  My-Custom-Header: my-custom-value
```


#### `bearer_token_file` [_bearer_token_file]

If defined, Metricbeat will read the contents of the file once at initialization and then use the value in an HTTP Authorization header.


#### `basepath` [_basepath]

An optional base path to be used in HTTP URIs. If defined, Metricbeat will insert this value as the first segment in the HTTP URI path.


#### `query` [_query]

An optional value to pass common query params in YAML. Instead of setting the query params within hosts values using the syntax `?key=value&key2&value2`, you can set it here like this:

```yaml
query:
  key: value
  key2: value2
  list:
  - 1.1
  - 2.95
  - -15
```

