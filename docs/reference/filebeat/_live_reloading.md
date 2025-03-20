---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/_live_reloading.html
---

# Live reloading [_live_reloading]

You can configure Filebeat to dynamically reload external configuration files when there are changes. This feature is available for input and module configurations that are loaded as [external configuration files](/reference/filebeat/filebeat-configuration-reloading.md). You cannot use this feature to reload the main `filebeat.yml` configuration file.

To configure this feature, you specify a path ([Glob](https://golang.org/pkg/path/filepath/#Glob)) to watch for configuration changes. When the files found by the Glob change, new inputs and/or modules are started and stopped according to changes in the configuration files.

This feature is especially useful in container environments where one container is used to tail logs for services running in other containers on the same host.

To enable dynamic config reloading, you specify the `path` and `reload` options under `filebeat.config.inputs` or `filebeat.config.modules` sections. For example:

```sh
filebeat.config.inputs:
  enabled: true
  path: configs/*.yml
  reload.enabled: true
  reload.period: 10s
```

`path`
:   A Glob that defines the files to check for changes.

`reload.enabled`
:   When set to `true`, enables dynamic config reload.

`reload.period`
:   Specifies how often the files are checked for changes. Do not set the `period` to less than 1s because the modification time of files is often stored in seconds. Setting the `period` to less than 1s will result in unnecessary overhead.

::::{note}
On systems with POSIX file permissions, all Beats configuration files are subject to ownership and file permission checks. For more information, see [Config File Ownership and Permissions](/reference/libbeat/config-file-permissions.md).
::::


