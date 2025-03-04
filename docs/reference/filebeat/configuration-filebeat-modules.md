---
navigation_title: "Modules"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/configuration-filebeat-modules.html
---

# Configure modules [configuration-filebeat-modules]


::::{note}
Using Filebeat modules is optional. You may decide to [configure inputs manually](/reference/filebeat/configuration-filebeat-options.md) if you’re using a log type that isn’t supported, or you want to use a different setup.
::::


Filebeat [modules](/reference/filebeat/filebeat-modules.md) provide a quick way to get started processing common log formats. They contain default configurations, {{es}} ingest pipeline definitions, and {{kib}} dashboards to help you implement and deploy a log monitoring solution.

You can configure modules in the `modules.d` directory (recommended), or in the Filebeat configuration file.

Before running Filebeat with modules enabled, make sure you also set up the environment to use {{kib}} dashboards. See [Quick start: installation and configuration](/reference/filebeat/filebeat-installation-configuration.md) for more information.

::::{note}
On systems with POSIX file permissions, all Beats configuration files are subject to ownership and file permission checks. For more information, see [Config File Ownership and Permissions](/reference/libbeat/config-file-permissions.md).
::::



## Configure modules in the `modules.d` directory [configure-modules-d-configs]

The `modules.d` directory contains default configurations for all the modules available in Filebeat. To enable or disable specific module configurations under `modules.d`, run the [`modules enable` or `modules disable`](/reference/filebeat/command-line-options.md#modules-command) command. For example:

:::::::{tab-set}

::::::{tab-item} DEB
```sh
filebeat modules enable nginx
```
::::::

::::::{tab-item} RPM
```sh
filebeat modules enable nginx
```
::::::

::::::{tab-item} MacOS
```sh
./filebeat modules enable nginx
```
::::::

::::::{tab-item} Linux
```sh
./filebeat modules enable nginx
```
::::::

::::::{tab-item} Windows
```sh
PS > .\filebeat.exe modules enable nginx
```
::::::

:::::::
The default configurations assume that your data is in the location expected for your OS and that the behavior of the module is appropriate for your environment. To change the default behavior, configure variable settings. For a list of available settings, see the documentation under [Modules](/reference/filebeat/filebeat-modules.md).

For advanced use cases, you can also [override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
You can enable modules at runtime by using the [--modules flag](/reference/filebeat/filebeat-modules.md). This is useful if you’re getting started and want to try things out. Any modules specified at the command line are loaded along with any modules that are enabled in the configuration file or `modules.d` directory. If there’s a conflict, the configuration specified at the command line is used.
::::



## Configure modules in the `filebeat.yml` file [configure-modules-config-file]

When possible, you should use the config files in the `modules.d` directory.

However, configuring [modules](/reference/filebeat/filebeat-modules.md) directly in the config file is a practical approach if you have upgraded from a previous version of Filebeat and don’t want to move your module configs to the `modules.d` directory. You can continue to configure modules in the `filebeat.yml` file, but you won’t be able to use the `modules` command to enable and disable configurations because the command requires the `modules.d` layout.

To enable specific modules in the `filebeat.yml` config file, add entries to the `filebeat.modules` list. Each entry in the list begins with a dash (-) and is followed by settings for that module.

The following example shows a configuration that runs the `nginx`,`mysql`, and `system` modules:

```yaml
filebeat.modules:
- module: nginx
  access:
  error:
- module: mysql
  slowlog:
- module: system
  auth:
```


