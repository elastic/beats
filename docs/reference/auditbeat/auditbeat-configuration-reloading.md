---
navigation_title: "Config file reloading"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/auditbeat-configuration-reloading.html
---

# Reload the configuration dynamically [auditbeat-configuration-reloading]


::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


You can configure Auditbeat to dynamically reload configuration files when there are changes. To do this, you specify a path ([glob](https://golang.org/pkg/path/filepath/#Glob)) to watch for module configuration changes. When the files found by the glob change, new modules are started/stopped according to changes in the configuration files.

To enable dynamic config reloading, you specify the `path` and `reload` options in the main `auditbeat.yml` config file. For example:

```sh
auditbeat.config.modules:
  path: ${path.config}/conf.d/*.yml
  reload.enabled: true
  reload.period: 10s
```

**`path`**
:   A glob that defines the files to check for changes.

**`reload.enabled`**
:   When set to `true`, enables dynamic config reload.

**`reload.period`**
:   Specifies how often the files are checked for changes. Do not set the `period` to less than 1s because the modification time of files is often stored in seconds. Setting the `period` to less than 1s will result in unnecessary overhead.

Each file found by the glob must contain a list of one or more module definitions. For example:

```yaml
- module: file_integrity
  paths:
  - /www/wordpress
  - /www/wordpress/wp-admin
  - /www/wordpress/wp-content
  - /www/wordpress/wp-includes
```

::::{note}
On systems with POSIX file permissions, all Beats configuration files are subject to ownership and file permission checks. If you encounter config loading errors related to file ownership, see [Config file ownership and permissions](/reference/libbeat/config-file-permissions.md).
::::


