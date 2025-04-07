---
navigation_title: "General settings"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/configuration-general-options.html
---

# Configure general settings [configuration-general-options]


You can specify settings in the `filebeat.yml` config file to control the general behavior of Filebeat. This includes:

* [Global options](#configuration-global-options) that control things like publisher behavior and the location of some files.
* [General options](#configuration-general) that are supported by all Elastic Beats.


## Global Filebeat configuration options [configuration-global-options]

These options are in the `filebeat` namespace.


### `registry.path` [_registry_path]

The root path of the registry.  If a relative path is used, it is considered relative to the data path. See the [Directory layout](/reference/filebeat/directory-layout.md) section for details. The default is `${path.data}/registry`.

```yaml
filebeat.registry.path: registry
```

::::{note}
The registry is only updated when new events are flushed and not on a predefined period. That means in case there are some states where the TTL expired, these are only removed when new events are processed.
::::



### `registry.file_permissions` [_registry_file_permissions]

The permissions mask to apply on registry data file. The default value is 0600. The permissions option must be a valid Unix-style file permissions mask expressed in octal notation. In Go, numbers in octal notation must start with 0.

The most permissive mask allowed is 0640. If a higher permissions mask is specified via this setting, it will be subject to an umask of 0027.

This option is not supported on Windows.

Examples:

* 0640: give read and write access to the file owner, and read access to members of the group associated with the file.
* 0600: give read and write access to the file owner, and no access to all others.

```yaml
filebeat.registry.file_permissions: 0600
```


### `registry.flush` [_registry_flush]

The timeout value that controls when registry entries are written to disk (flushed). When an unwritten update exceeds this value, it triggers a write to disk. When `registry.flush` is set to 0s, the registry is written to disk after each batch of events has been published successfully. The default value is 1s.

::::{note}
The registry is always updated when Filebeat shuts down normally. After an abnormal shutdown, the registry will not be up-to-date if the `registry.flush` value is >0s. Filebeat will send published events again (depending on values in the last updated registry file).
::::


::::{note}
Filtering out a huge number of logs can cause many registry updates, slowing down processing. Setting `registry.flush` to a value >0s reduces write operations, helping Filebeat process more events.
::::



### `registry.migrate_file` [_registry_migrate_file]

Prior to Filebeat 7.0 the registry is stored in a single file. When you upgrade to 7.0, Filebeat will automatically migrate the old Filebeat 6.x registry file to use the new directory format. Filebeat looks for the file in the location specified by `filebeat.registry.path`. If you changed the path while upgrading, set `filebeat.registry.migrate_file` to point to the old registry file.

```yaml
filebeat.registry.path: ${path.data}/registry
filebeat.registry.migrate_file: /path/to/old/registry_file
```

The registry will be migrated to the new location only if a registry using the directory format does not already exist.


### `config_dir` [_config_dir]

[6.0.0]

The full path to the directory that contains additional input configuration files. Each configuration file must end with `.yml`. Each config file must also specify the full Filebeat config hierarchy even though only the `inputs` part of each file is processed. All global options, such as `registry_file`, are ignored.

The `config_dir` option MUST point to a directory other than the directory where the main Filebeat config file resides.

If the specified path is not absolute, it is considered relative to the configuration path. See the [Directory layout](/reference/filebeat/directory-layout.md) section for details.

```yaml
filebeat.config_dir: path/to/configs
```


### `shutdown_timeout` [shutdown-timeout]

How long Filebeat waits on shutdown for the publisher to finish sending events before Filebeat shuts down.

By default, this option is disabled, and Filebeat does not wait for the publisher to finish sending events before shutting down. This means that any events sent to the output, but not acknowledged before Filebeat shuts down, are sent again when you restart Filebeat. For more details about how this works, see [How does Filebeat ensure at-least-once delivery?](/reference/filebeat/how-filebeat-works.md#at-least-once-delivery).

You can configure the `shutdown_timeout` option to specify the maximum amount of time that Filebeat waits for the publisher to finish sending events before shutting down. If all events are acknowledged before `shutdown_timeout` is reached, Filebeat will shut down.

There is no recommended setting for this option because determining the correct value for `shutdown_timeout` depends heavily on the environment in which Filebeat is running and the current state of the output.

Example configuration:

```yaml
filebeat.shutdown_timeout: 5s
```


## General configuration options [configuration-general]


These options are supported by all Elastic Beats. Because they are common options, they are not namespaced.

Here is an example configuration:

```yaml
name: "my-shipper"
tags: ["service-X", "web-tier"]
```


### `name` [_name_2]

The name of the Beat. If this option is empty, the `hostname` of the server is used. The name is included as the `agent.name` field in each published transaction. You can use the name to group all transactions sent by a single Beat.

Example:

```yaml
name: "my-shipper"
```


### `tags` [_tags_30]

A list of tags that the Beat includes in the `tags` field of each published transaction. Tags make it easy to group servers by different logical properties. For example, if you have a cluster of web servers, you can add the "webservers" tag to the Beat on each server, and then use filters and queries in the Kibana web interface to get visualisations for the whole group of servers.

Example:

```yaml
tags: ["my-service", "hardware", "test"]
```


### `fields` [libbeat-configuration-fields]

Optional fields that you can specify to add additional information to the output. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true.

Example:

```yaml
fields: {project: "myproject", instance-id: "574734885120952459"}
```


### `fields_under_root` [_fields_under_root_2]

If this option is set to true, the custom [fields](#libbeat-configuration-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names, then the custom fields overwrite the other fields.

Example:

```yaml
fields_under_root: true
fields:
  instance_id: i-10a64379
  region: us-east-1
```


### `processors` [_processors_30]

A list of processors to apply to the data generated by the beat.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


### `max_procs` [_max_procs]

Sets the maximum number of CPUs that can be executing simultaneously. The default is the number of logical CPUs available in the system.


### `timestamp.precision` [_timestamp_precision]

Configure the precision of all timestamps. By default it is set to millisecond. Available options: millisecond, microsecond, nanosecond

