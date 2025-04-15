---
navigation_title: "Project paths"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/configuration-path.html
---

# Configure project paths [configuration-path]


The `path` section of the `winlogbeat.yml` config file contains configuration options that define where Winlogbeat looks for its files. For example, Winlogbeat looks for the Elasticsearch template file in the configuration path and writes log files in the logs path. Winlogbeat looks for its registry files in the data path.

Please see the [Directory layout](/reference/winlogbeat/directory-layout.md) section for more details.

Here is an example configuration:

```yaml
path.home: /usr/share/beat
path.config: /etc/beat
path.data: /var/lib/beat
path.logs: /var/log/
```

Note that it is possible to override these options by using command line flags.


## Configuration options [_configuration_options_2]

You can specify the following options in the `path` section of the `winlogbeat.yml` config file:


### `home` [_home]

The home path for the Winlogbeat installation. This is the default base path for all other path settings and for miscellaneous files that come with the distribution (for example, the sample dashboards). If not set by a CLI flag or in the configuration file, the default for the home path is the location of the Winlogbeat binary.

Example:

```yaml
path.home: /usr/share/beats
```


### `config` [_config]

The configuration path for the Winlogbeat installation. This is the default base path for configuration files, including the main YAML configuration file and the Elasticsearch template file. If not set by a CLI flag or in the configuration file, the default for the configuration path is the home path.

Example:

```yaml
path.config: /usr/share/beats/config
```


### `data` [_data]

The data path for the Winlogbeat installation. This is the default base path for all the files in which Winlogbeat needs to store its data. If not set by a CLI flag or in the configuration file, the default for the data path is a `data` subdirectory inside the home path.

Example:

```yaml
path.data: /var/lib/beats
```

::::{tip}
When running multiple Winlogbeat instances on the same host, make sure they each have a distinct `path.data` value.
::::



### `logs` [_logs]

The logs path for a Winlogbeat installation. This is the default location for Winlogbeat’s log files. If not set by a CLI flag or in the configuration file, the default for the logs path is a `logs` subdirectory inside the home path.

Example:

```yaml
path.logs: /var/log/beats
```

