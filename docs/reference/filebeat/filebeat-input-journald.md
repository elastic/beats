---
navigation_title: "journald"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-journald.html
---

# Journald input [filebeat-input-journald]

[`journald`](https://www.freedesktop.org/software/systemd/man/systemd-journald.service.html) is a system service that collects and stores logging data. The `journald` input reads this log data and the metadata associated with it. To read this log data Filebeat calls `journalctl` to read from the journal, therefore Filebeat needs permission to execute `journalctl`.

:::{warning}
The Wolfi-based Docker image does not contain the `journalctl` binary and the `journald` input type cannot be used with it.
:::

:::{important}
When using the Journald input from a Docker container, make sure the
`journalctl` binary in the container is compatible with your
Systemd/journal version. To get the version of the `journalctl` binary
in Filebeat's image run the following, adjusting the image name/tag
according to the version that you are running:


```sh
docker run --rm -it --entrypoint "journalctl" docker.elastic.co/beats/filebeat-wolfi:<VERSION> --version
```
:::

If the `journalctl` process exits unexpectedly the journald input will terminate with an error and Filebeat will need to be restarted to start reading from the journal again.

The simplest configuration example is one that reads all logs from the default journal.

```yaml
filebeat.inputs:
- type: journald
  id: everything
```

You may wish to have separate inputs for each service. You can use `include_matches` to specify filtering expressions. A good way to list the [journald fields](https://www.freedesktop.org/software/systemd/man/systemd.journal-fields.html) that are available for filtering messages is to run `journalctl -o json` to output logs and metadata as JSON. This example collects logs from the `vault.service` systemd unit.

```yaml
filebeat.inputs:
- type: journald
  id: service-vault
  include_matches.match:
    - _SYSTEMD_UNIT=vault.service
```

This example collects kernel logs where the message begins with `iptables`. Note that `include_matches` is more efficient than Beat processors because that are applied before the data is passed to the Filebeat so prefer them where possible.

```yaml
filebeat.inputs:
- type: journald
  id: iptables
  include_matches.match:
    - _TRANSPORT=kernel
  processors:
    - drop_event:
        when.not.regexp.message: '^iptables'
```

Each example adds the `id` for the input to ensure the cursor is persisted to the registry with a unique ID. The ID should be unique among journald inputs. If you don’t specify and `id` then one is created for you by hashing the configuration. So when you modify the config this will result in a new ID and a fresh cursor.

## Configuration options [filebeat-input-journald-options]

The `journald` input supports the following configuration options plus the [Common options](#filebeat-input-journald-common-options) described later.


### `id` [filebeat-input-journald-id]

An unique identifier for the input. By providing a unique `id` you can operate multiple inputs on the same journal. This allows each input’s cursor to be persisted independently in the registry file. Each journald input must have an unique ID.

```yaml
filebeat.inputs:
- type: journald
  id: consul.service
  include_matches.match:
    - _SYSTEMD_UNIT=consul.service

- type: journald
  id: vault.service
  include_matches.match:
    - _SYSTEMD_UNIT=vault.service
```


### `paths` [filebeat-input-journald-paths]

A list of paths that will be crawled and fetched. Each path can be a directory path (to collect events from all journals in a directory), or a file path. If you specify a directory, Filebeat merges all journals under the directory into a single journal and reads them.

If no paths are specified, Filebeat reads from the default journal.


### `seek` [filebeat-input-journald-seek]

The position to start reading the journal from. Valid settings are:

* `head`: Starts reading at the beginning of the journal. After a restart, Filebeat resends all log messages in the journal.
* `tail`: Starts reading at the end of the journal. This means that no events will be sent until a new message is written.
* `since`: Use the `since` option to determine where to start reading from.

Regardless of the value of `seek` if Filebeat has a state (cursor) for this input, the `seek` value is ignored and the current cursor is used. To reset the cursor, just change the `id` of the input, this will start from a fresh state.


### `since` [filebeat-input-journald-since]

A time offset from the current time to start reading from. To use `since`, `seek` option must be set to `since`.

This example demonstrates how to resume from the persisted cursor when it exists, or otherwise begin reading logs from the last 24 hours.

```yaml
seek: since
since: -24h
```


### `units` [filebeat-input-journald-units]

Iterate only the entries of the units specified in this option. The iterated entries include messages from the units, messages about the units by authorized daemons and coredumps. However, it does not match systemd user units.


### `syslog_identifiers` [filebeat-input-journald-syslog-identifiers]

Read only the entries with the selected syslog identifiers.


### `transports` [filebeat-input-journald-transports]

Collect the messages using the specified transports. Example: syslog.

Valid transports:

* audit: messages from the kernel audit subsystem
* driver: internally generated messages
* syslog: messages received via the local syslog socket with the syslog protocol
* journal: messages received via the native journal protocol
* stdout: messages from a service’s standard output or error output
* kernel: messages from the kernel


### `facilities` [filebeat-input-journald-facilities]

Filter entries by facilities, facilities must be specified using their numeric code.


### `include_matches` [filebeat-input-journald-include-matches]

A collection of filter expressions used to match fields. The format of the expression is `field=value` or `+` representing disjunction (i.e. logical OR). Filebeat fetches all events that exactly match the expressions. Pattern matching is not supported.

When `+` is used, it will cause all matches before and after to be combined in a disjunction (i.e. logical OR).

If you configured a filter expression, only entries with this field set will be iterated by the journald reader of Filebeat. If the filter expressions apply to different fields, only entries with all fields set will be iterated. If they apply to the same fields, only entries where the field takes one of the specified values will be iterated.

`match`: List of filter expressions to match fields.

Please note that these expressions are limited. You can build complex filtering, but full logical expressions are not supported.

The following include matches configuration will ingest entries that contain `journald.process.name: systemd` and `systemd.transport: syslog`.

```yaml
include_matches:
  match:
    - "journald.process.name=systemd"
    - "systemd.transport=syslog"
```

The following include matches configuration will ingest entries that contain `systemd.transport: systemd` or `systemd.transport: kernel`.

```yaml
include_matches:
  match:
    - "systemd.transport=kernel"
    - "systemd.transport=syslog"
```

The following include matches configuration is the equivalent of the following logical expression:
 
 ```
 A=a OR (B=b AND C=c) OR (D=d AND B=1)
 ```
 
```yaml
 include_matches:
   match:
     - A=a
     - +
     - B=b
     - C=c
     - +
     - B=1
```
 
`include_matches` translates to `journalctl` `MATCHES`, its [documentation](https://www.man7.org/linux/man-pages/man1/journalctl.1.html)  is not clear about how multiple disjunctions are handled. The previous example was tested with journalctl version 257.

To reference fields, use one of the following:

* The field name used by the systemd journal. For example, `CONTAINER_TAG=redis`.
* The [translated field name](#filebeat-input-journald-translated-fields) used by Filebeat. For example, `container.image.tag=redis`. Filebeat does not translate all fields from the journal. For custom fields, use the name specified in the systemd journal.


#### `parsers` [_parsers_2]

This option expects a list of parsers that the entry has to go through.

Available parsers:

* `multiline`
* `ndjson`
* `container`
* `syslog`
* `include_message`

In this example, Filebeat is reading multiline messages that consist of 3 lines and are encapsulated in single-line JSON objects. The multiline message is stored under the key `msg`.

```yaml
filebeat.inputs:
- type: journald
  ...
  parsers:
    - ndjson:
        target: ""
        message_key: msg
    - multiline:
        type: count
        count_lines: 3
```

See the available parser settings in detail below.


#### `multiline` [_multiline_4]

Options that control how Filebeat deals with log messages that span multiple lines. See [Multiline messages](/reference/filebeat/multiline-examples.md) for more information about configuring multiline options.


#### `ndjson` [filebeat-input-journald-ndjson]

These options make it possible for Filebeat to decode logs structured as JSON messages. Filebeat processes the entry by line, so the JSON decoding only works if there is one JSON object per message.

The decoding happens before line filtering. You can combine JSON decoding with filtering if you set the `message_key` option. This can be helpful in situations where the application logs are wrapped in JSON objects, like when using Docker.

Example configuration:

```yaml
- ndjson:
    target: ""
    add_error_key: true
    message_key: log
```

**`target`**
:   The name of the new JSON object that should contain the parsed key value pairs. If you leave it empty, the new keys will go under root.

**`overwrite_keys`**
:   Values from the decoded JSON object overwrite the fields that Filebeat normally adds (type, source, offset, etc.) in case of conflicts. Disable it if you want to keep previously added values.

**`expand_keys`**
:   If this setting is enabled, Filebeat will recursively de-dot keys in the decoded JSON, and expand them into a hierarchical object structure. For example, `{"a.b.c": 123}` would be expanded into `{"a":{"b":{"c":123}}}`. This setting should be enabled when the input is produced by an [ECS logger](https://github.com/elastic/ecs-logging).

**`add_error_key`**
:   If this setting is enabled, Filebeat adds an "error.message" and "error.type: json" key in case of JSON unmarshalling errors or when a `message_key` is defined in the configuration but cannot be used.

**`message_key`**
:   An optional configuration setting that specifies a JSON key on which to apply the line filtering and multiline settings. If specified the key must be at the top level in the JSON object and the value associated with the key must be a string, otherwise no filtering or multiline aggregation will occur.

**`document_id`**
:   Option configuration setting that specifies the JSON key to set the document id. If configured, the field will be removed from the original JSON document and stored in `@metadata._id`

**`ignore_decoding_error`**
:   An optional configuration setting that specifies if JSON decoding errors should be logged or not. If set to true, errors will not be logged. The default is false.


#### `container` [_container]

Use the `container` parser to extract information from  containers log files. It parses lines into common message lines, extracting timestamps too.

**`stream`**
:   Reads from the specified streams only: `all`, `stdout` or `stderr`. The default is `all`.

**`format`**
:   Use the given format when parsing logs: `auto`, `docker` or `cri`. The default is `auto`, it will automatically detect the format. To disable autodetection set any of the other options.

The following snippet configures Filebeat to read the `stdout` stream from all containers under the default Kubernetes logs path:

```yaml
  parsers:
    - container:
        stream: stdout
```


#### `syslog` [_syslog_2]

The `syslog` parser parses RFC 3146 and/or RFC 5424 formatted syslog messages.

The supported configuration options are:

**`format`**
:   (Optional) The syslog format to use, `rfc3164`, or `rfc5424`. To automatically detect the format from the log entries, set this option to `auto`. The default is `auto`.

**`timezone`**
:   (Optional) IANA time zone name(e.g. `America/New York`) or a fixed time offset (e.g. +0200) to use when parsing syslog timestamps that do not contain a time zone. `Local` may be specified to use the machine’s local time zone. Defaults to `Local`.

**`log_errors`**
:   (Optional) If `true` the parser will log syslog parsing errors. Defaults to `false`.

**`add_error_key`**
:   (Optional) If this setting is enabled, the parser adds or appends to an `error.message` key with the parsing error that was encountered. Defaults to `true`.

Example configuration:

```yaml
- syslog:
    format: rfc3164
    timezone: America/Chicago
    log_errors: true
    add_error_key: true
```

**Timestamps**

The RFC 3164 format accepts the following forms of timestamps:

* Local timestamp (`Mmm dd hh:mm:ss`):

    * `Jan 23 14:09:01`

* RFC-3339*:

    * `2003-10-11T22:14:15Z`
    * `2003-10-11T22:14:15.123456Z`
    * `2003-10-11T22:14:15-06:00`
    * `2003-10-11T22:14:15.123456-06:00`


**Note**: The local timestamp (for example, `Jan 23 14:09:01`) that accompanies an RFC 3164 message lacks year and time zone information. The time zone will be enriched using the `timezone` configuration option, and the year will be enriched using the Filebeat system’s local time (accounting for time zones). Because of this, it is possible for messages to appear in the future. An example of when this might happen is logs generated on December 31 2021 are ingested on January 1 2022. The logs would be enriched with the year 2022 instead of 2021.

The RFC 5424 format accepts the following forms of timestamps:

* RFC-3339:

    * `2003-10-11T22:14:15Z`
    * `2003-10-11T22:14:15.123456Z`
    * `2003-10-11T22:14:15-06:00`
    * `2003-10-11T22:14:15.123456-06:00`


Formats with an asterisk (*) are a non-standard allowance.


#### `include_message` [_include_message_2]

Use the `include_message` parser to filter messages in the parsers pipeline. Messages that match the provided pattern are passed to the next parser, the others are dropped.

You should use `include_message` instead of `include_lines` if you would like to control when the filtering happens. `include_lines` runs after the parsers, `include_message` runs in the parsers pipeline.

**`patterns`**
:   List of regexp patterns to match.

This example shows you how to include messages that start with the string ERR or WARN:

```yaml
  parsers:
    - include_message.patterns: ["^ERR", "^WARN"]
```


## Translated field names [filebeat-input-journald-translated-fields]

You can use the following translated names in filter expressions to reference journald fields:

**Journald field name**
:   **Translated name**

`COREDUMP_UNIT`
:   `journald.coredump.unit`

`COREDUMP_USER_UNIT`
:   `journald.coredump.user_unit`

`OBJECT_AUDIT_LOGINUID`
:   `journald.object.audit.login_uid`

`OBJECT_AUDIT_SESSION`
:   `journald.object.audit.session`

`OBJECT_CMDLINE`
:   `journald.object.cmd`

`OBJECT_COMM`
:   `journald.object.name`

`OBJECT_EXE`
:   `journald.object.executable`

`OBJECT_GID`
:   `journald.object.gid`

`OBJECT_PID`
:   `journald.object.pid`

`OBJECT_SYSTEMD_OWNER_UID`
:   `journald.object.systemd.owner_uid`

`OBJECT_SYSTEMD_SESSION`
:   `journald.object.systemd.session`

`OBJECT_SYSTEMD_UNIT`
:   `journald.object.systemd.unit`

`OBJECT_SYSTEMD_USER_UNIT`
:   `journald.object.systemd.user_unit`

`OBJECT_UID`
:   `journald.object.uid`

`_AUDIT_LOGINUID`
:   `process.audit.login_uid`

`_AUDIT_SESSION`
:   `process.audit.session`

`_BOOT_ID`
:   `host.boot_id`

`_CAP_EFFECTIVE`
:   `process.capabilites`

`_CMDLINE`
:   `process.cmd`

`_CODE_FILE`
:   `journald.code.file`

`_CODE_FUNC`
:   `journald.code.func`

`_CODE_LINE`
:   `journald.code.line`

`_COMM`
:   `process.name`

`_EXE`
:   `process.executable`

`_GID`
:   `process.uid`

`_HOSTNAME`
:   `host.name`

`_KERNEL_DEVICE`
:   `journald.kernel.device`

`_KERNEL_SUBSYSTEM`
:   `journald.kernel.subsystem`

`_MACHINE_ID`
:   `host.id`

`_MESSAGE`
:   `message`

`_PID`
:   `process.pid`

`_PRIORITY`
:   `logs.syslog.priority`

`_SYSLOG_FACILITY`
:   `logs.syslog.facility.code`

`_SYSLOG_IDENTIFIER`
:   `logs.syslog.identifier.appname`

`_SYSLOG_PID`
:   `log.syslog.procid`

`_SYSTEMD_CGROUP`
:   `systemd.cgroup`

`_SYSTEMD_INVOCATION_ID`
:   `systemd.invocation_id`

`_SYSTEMD_OWNER_UID`
:   `systemd.owner_uid`

`_SYSTEMD_SESSION`
:   `systemd.session`

`_SYSTEMD_SLICE`
:   `systemd.slice`

`_SYSTEMD_UNIT`
:   `systemd.unit`

`_SYSTEMD_USER_SLICE`
:   `systemd.user_slice`

`_SYSTEMD_USER_UNIT`
:   `systemd.user_unit`

`_TRANSPORT`
:   `systemd.transport`

`_UDEV_DEVLINK`
:   `journald.kernel.device_symlinks`

`_UDEV_DEVNODE`
:   `journald.kernel.device_node_path`

`_UDEV_SYSNAME`
:   `journald.kernel.device_name`

`_UID`
:   `process.uid`

The following translated fields for [Docker](https://docs.docker.com/config/containers/logging/journald/) are also available:

`CONTAINER_ID_FULL`
:   `container.id`

`CONTAINER_NAME`
:   `container.name`

`IMAGE_NAME`
:   `container.image.name`

If `CONTAINER_PARTIAL_MESSAGE` is present and it is true, then the tag `partial_message` is added to the final event.

## Common options [filebeat-input-journald-common-options]

The following configuration options are supported by all inputs.


#### `enabled` [_enabled_14]

Use the `enabled` option to enable and disable inputs. By default, enabled is set to true.


#### `tags` [_tags_14]

A list of tags that Filebeat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
filebeat.inputs:
- type: journald
  . . .
  tags: ["json"]
```


#### `fields` [filebeat-input-journald-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
filebeat.inputs:
- type: journald
  . . .
  fields:
    app_id: query_engine_12
```


#### `fields_under_root` [fields-under-root-journald]

If this option is set to true, the custom [fields](#filebeat-input-journald-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


#### `processors` [_processors_14]

A list of processors to apply to the input data.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


#### `pipeline` [_pipeline_14]

The ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::


::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::



#### `keep_null` [_keep_null_14]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `index` [_index_14]

If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.11.01"`.


#### `publisher_pipeline.disable_host` [_publisher_pipeline_disable_host_14]

By default, all events contain `host.name`. This option can be set to `true` to disable the addition of this field to all events. The default value is `false`.


