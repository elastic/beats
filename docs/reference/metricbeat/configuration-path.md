---
navigation_title: "Project paths"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/configuration-path.html
---

# Configure project paths [configuration-path]


The `path` section of the `metricbeat.yml` config file contains configuration options that define where Metricbeat looks for its files. For example, Metricbeat looks for the Elasticsearch template file in the configuration path and writes log files in the logs path.

Please see the [Directory layout](/reference/metricbeat/directory-layout.md) section for more details.

Here is an example configuration:

```yaml
path.home: /usr/share/beat
path.config: /etc/beat
path.data: /var/lib/beat
path.logs: /var/log/
```

Note that it is possible to override these options by using command line flags.


## Configuration options [_configuration_options]

You can specify the following options in the `path` section of the `metricbeat.yml` config file:


### `home` [_home]

The home path for the Metricbeat installation. This is the default base path for all other path settings and for miscellaneous files that come with the distribution (for example, the sample dashboards). If not set by a CLI flag or in the configuration file, the default for the home path is the location of the Metricbeat binary.

Example:

```yaml
path.home: /usr/share/beats
```


### `config` [_config]

The configuration path for the Metricbeat installation. This is the default base path for configuration files, including the main YAML configuration file and the Elasticsearch template file. If not set by a CLI flag or in the configuration file, the default for the configuration path is the home path.

Example:

```yaml
path.config: /usr/share/beats/config
```


### `data` [_data]

The data path for the Metricbeat installation. This is the default base path for all the files in which Metricbeat needs to store its data. If not set by a CLI flag or in the configuration file, the default for the data path is a `data` subdirectory inside the home path.

Example:

```yaml
path.data: /var/lib/beats
```

::::{tip}
When running multiple Metricbeat instances on the same host, make sure they each have a distinct `path.data` value.
::::



### `logs` [_logs]

The logs path for a Metricbeat installation. This is the default location for Metricbeat’s log files. If not set by a CLI flag or in the configuration file, the default for the logs path is a `logs` subdirectory inside the home path.

Example:

```yaml
path.logs: /var/log/beats
```


### `system.hostfs` [_system_hostfs]

Specifies the mount point of the host’s filesystem for use in monitoring a host. This can either be set in the config, or with the `--system.hostfs` CLI flag. This is used for cgroup self-monitoring.

This is also used by the system module to read files from `/proc` and `/sys`. This option is deprecated and will be removed in a future release. To set the filesystem root, use the `hostfs` flag inside the module-level config.

Example:

```yaml
system.hostfs: /mount/rootfs
```

