---
navigation_title: "Config file loading"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-configuration-reloading.html
---

# Load external configuration files [metricbeat-configuration-reloading]


Metricbeat can load external configuration files for modules, which allows you to separate your configuration into multiple smaller configuration files. To use this, you specify the `path` option under `metricbeat.config.modules` in the main `metricbeat.yml` configuration file. By default, Metricbeat loads the module configurations enabled in the [`modules.d`](/reference/metricbeat/configuration-metricbeat.md#configure-modules-d-configs) directory. For example:

```yaml
metricbeat.config.modules:
  path: ${path.config}/modules.d/*.yml
```

`path`
:   A Glob that defines the files to check for changes.

    This setting must point to the `modules.d` directory if you want to use the [`modules`](/reference/metricbeat/command-line-options.md#modules-command) command to enable and disable module configurations.


Each file found by the Glob must contain a list of one or more module definitions. For example:

```yaml
- module: system
  metricsets: ["cpu"]
  enabled: false
  period: 1s

- module: system
  metricsets: ["network"]
  enabled: true
  period: 10s
```

::::{note}
On systems with POSIX file permissions, all Beats configuration files are subject to ownership and file permission checks. For more information, see [Config File Ownership and Permissions](/reference/libbeat/config-file-permissions.md).
::::



