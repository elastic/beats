---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/_live_reloading.html
---

# Live reloading [_live_reloading]

You can configure Metricbeat to dynamically reload configuration files when there are changes. To do this, you specify a path ([Glob](https://golang.org/pkg/path/filepath/#Glob)) to watch for module configuration changes. When the files found by the Glob change, new modules are started/stopped according to changes in the configuration files.

This feature is especially useful in container environments where one container is used to monitor all services running in other containers on the same host. Because new containers appear and disappear dynamically, you may need to change the Metricbeat configuration frequently to specify which modules are needed and which hosts must be monitored.

To enable dynamic config reloading, you specify the `path` and `reload` options under `metricbeat.config.modules` in the main `metricbeat.yml` config file. For example:

```yaml
metricbeat.config.modules:
  path: ${path.config}/modules.d/*.yml
  reload.enabled: true
  reload.period: 10s
```

`path`
:   A Glob that defines the files to check for changes.

    This setting must point to the `modules.d` directory if you want to use the [`modules`](/reference/metricbeat/command-line-options.md#modules-command) command to enable and disable module configurations.


`reload.enabled`
:   When set to `true`, enables dynamic config reload.

`reload.period`
:   Specifies how often the files are checked for changes. Do not set the `period` to less than 1s because the modification time of files is often stored in seconds. Setting the `period` to less than 1s will result in unnecessary overhead.

::::{note}
On systems with POSIX file permissions, all Beats configuration files are subject to ownership and file permission checks. For more information, see [Config File Ownership and Permissions](/reference/libbeat/config-file-permissions.md).
::::


