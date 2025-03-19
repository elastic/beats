---
navigation_title: "Override configuration settings"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/override-filebeat-config-settings.html
---

# Override configuration settings at the command line [override-filebeat-config-settings]


::::{note}
If you’re running Filebeat as a service, you can’t specify command-line flags. To specify flags, start Filebeat in the foreground.
::::


You can override any configuration setting from the command line by using flags:

`-E, --E "SETTING_NAME=VALUE"`
:   Overrides a specific configuration setting.

`-M, --M "VAR_NAME=VALUE"`
:   Overrides the default configuration for a module.

You can specify multiple overrides. Overrides are applied to the currently running Filebeat process. The Filebeat configuration file is not changed.


## Example: override configuration file settings [example-override-config]

The following configuration sends logging output to files:

```sh
logging.level: info
logging.to_files: true
logging.files:
  path: /var/log/filebeat
  name: filebeat
  keepfiles: 7
  permissions: 0640
```

To override the logging level and send logging output to standard error instead of a file, use the `-E` flag when you run Filebeat:

```sh
-E "logging.to_files=false" -E "logging.to_stderr=true" -E "logging.level=error"
```


## Example: override module settings [example-override-module-setting]

The following configuration sets the path to Nginx access logs:

```yaml
- module: nginx
  access:
    var.paths: ["/var/log/nginx/access.log*"]
```

To override the `var.paths` setting from the command line, use the `-M` flag when you run Filebeat. The variable name must include the module and fileset name. For example:

```sh
-M "nginx.access.var.paths=[/path/to/log/nginx/access.log*]"
```

You can specify multiple overrides. Each override must start with `-M`.

For information about specific variables that you can set for each fileset, see the documentation under [Modules](/reference/filebeat/filebeat-modules.md).

