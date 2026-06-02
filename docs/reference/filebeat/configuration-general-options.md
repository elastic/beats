---
navigation_title: "General settings"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/configuration-general-options.html
applies_to:
  stack: ga
  serverless: ga
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


### `registry.backend` [_registry_backend]

::::{warning}
The bbolt backend is **experimental** and may change or be removed in future releases. Do not use in production without understanding the risks.
::::

The storage backend used for the registry. Supported values:

- `memlog` (default): An in-memory log with periodic disk flushing. This is the original backend and is well-tested.
- `bbolt`: A [bbolt (BoltDB)](https://github.com/etcd-io/bbolt) database for persistent on-disk storage with support for compaction and TTL-based entry cleanup.

```yaml
filebeat.registry.backend: memlog
```

When set to `bbolt`, the database files are stored under the directory specified by `registry.path`. The bbolt-specific settings are configured under `registry.bbolt`.


### `registry.bbolt.timeout` [_registry_bbolt_timeout]

The amount of time to wait to obtain a file lock on the bbolt database file. Default: `1s`.

```yaml
filebeat.registry.bbolt.timeout: 1s
```


### `registry.bbolt.fsync` [_registry_bbolt_fsync]

Controls whether the database calls `fdatasync()` after each write transaction commit. Default: `false`.

When set to `false`, writes are buffered by the operating system and flushed to disk lazily. This provides significantly higher write throughput but means that recent writes can be lost if the machine crashes (power failure, kernel panic, etc.) because data may still be in the OS page cache. A normal Filebeat shutdown is not affected — the database is closed cleanly and all data is flushed.

When set to `true`, every write transaction is synced to disk before returning. This guarantees durability at the cost of reduced write throughput, as each `Set` call must wait for the disk I/O to complete.

For most deployments, the default (`false`) is recommended. The registry tracks file offsets, so the worst case after an unclean shutdown is re-reading a small amount of log data that was already sent but whose offset was not yet persisted to disk.

```yaml
filebeat.registry.bbolt.fsync: false
```


### `registry.bbolt.compaction.on_start` [_registry_bbolt_compaction_on_start]

If `true`, database compaction runs every time Filebeat starts. Compaction rewrites the database file to reclaim unused disk space. Default: `false`.

```yaml
filebeat.registry.bbolt.compaction.on_start: false
```


### `registry.bbolt.compaction.max_transaction_size` [_registry_bbolt_compaction_max_transaction_size]

The maximum number of items processed per transaction during compaction and retention cleanup. Limiting the transaction size prevents a single large transaction from consuming excessive memory or holding a write lock for too long. A value of `0` disables batching, processing all items in a single transaction. Default: `65536`.

```yaml
filebeat.registry.bbolt.compaction.max_transaction_size: 65536
```


### `registry.bbolt.compaction.cleanup_on_start` [_registry_bbolt_compaction_cleanup_on_start]

If `true`, leftover temporary files from a previous compaction that was interrupted (for example, by a crash) are removed when Filebeat starts. Default: `false`.

```yaml
filebeat.registry.bbolt.compaction.cleanup_on_start: false
```


### `registry.bbolt.retention.ttl` [_registry_bbolt_retention_ttl]

How long entries are kept in the store before being removed. A zero value disables TTL-based removal. Expired entries become invisible to reads immediately, but are only physically deleted from disk when `registry.bbolt.retention.interval` is also set to a positive value. Default: `0` (disabled).

```yaml
filebeat.registry.bbolt.retention.ttl: 0
```


### `registry.bbolt.retention.interval` [_registry_bbolt_retention_interval]

How often to remove expired entries from disk. Only effective when `registry.bbolt.retention.ttl` is also set to a positive value. A zero value disables periodic removal. Default: `0` (disabled).

```yaml
filebeat.registry.bbolt.retention.interval: 0
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
