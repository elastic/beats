---
navigation_title: "Config file loading"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-configuration-reloading.html
---

# Load external configuration files [filebeat-configuration-reloading]


Filebeat can load external configuration files for inputs and modules, allowing you to separate your configuration into multiple smaller configuration files. See the [Input config](#load-input-config) and the [Module config](#load-module-config) sections for details.

::::{note}
On systems with POSIX file permissions, all Beats configuration files are subject to ownership and file permission checks. For more information, see [Config File Ownership and Permissions](/reference/libbeat/config-file-permissions.md).
::::



## Input config [load-input-config]

For input configurations, you specify the `path` option in the `filebeat.config.inputs` section of the `filebeat.yml` file. For example:

```sh
filebeat.config.inputs:
  enabled: true
  path: inputs.d/*.yml
```

Each file found by the `path` Glob must contain a list of one or more input definitions.

::::{tip}
The first line of each external configuration file must be an input definition that starts with `- type`. Make sure you omit the line `filebeat.config.inputs` from this file. All [`input type configuration options`](/reference/filebeat/configuration-filebeat-options.md#filebeat-input-types) must be specified within each external configuration file.  Specifying these configuration options at the global `filebeat.config.inputs` level is not supported.
::::


Example external configuration file:

```yaml
- type: filestream
  id: first
  paths:
    - /var/log/mysql.log
  prospector.scanner.check_interval: 10s

- type: filestream
  id: second
  paths:
    - /var/log/apache.log
  prospector.scanner.check_interval: 5s
```

::::{warning}
It is critical that two running inputs DO NOT have overlapping file paths defined. If more than one input harvests the same file at the same time, it can lead to unexpected behavior.
::::



## Module config [load-module-config]

For module configurations, you specify the `path` option in the `filebeat.config.modules` section of the `filebeat.yml` file. By default, Filebeat loads the module configurations enabled in the [`modules.d`](/reference/filebeat/configuration-filebeat-modules.md#configure-modules-d-configs) directory. For example:

```sh
filebeat.config.modules:
  enabled: true
  path: ${path.config}/modules.d/*.yml
```

The `path` setting must point to the `modules.d` directory if you want to use the [`modules`](/reference/filebeat/command-line-options.md#modules-command) command to enable and disable module configurations.

Each file found by the Glob must contain a list of one or more module definitions.

::::{tip}
The first line of each external configuration file must be a module definition that starts with `- module`. Make sure you omit the line `filebeat.config.modules` from this file.
::::


For example:

```yaml
- module: apache
  access:
    enabled: true
    var.paths: [/var/log/apache2/access.log*]
  error:
    enabled: true
    var.paths: [/var/log/apache2/error.log*]
```


