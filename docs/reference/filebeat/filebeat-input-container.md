---
navigation_title: "Container"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-container.html
---

# Container input [filebeat-input-container]

::::{warning}
The container input is just a preset for the [`log`](/reference/filebeat/filebeat-input-log.md) input. The `log` input is deprecated in version 7.16 and disabled in version 9.0.

Please use the the [`filestream`](/reference/filebeat/filebeat-input-filestream.md) input with its [`container`](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-parsers-container) parser instead. Follow [our official guide](/reference/filebeat/migrate-to-filestream.md) to migrate existing `log`/`container` inputs to `filestream` inputs.

After deprecation it’s possible to use this input type (e.g. for migration to `filestream`) only in combination with the `allow_deprecated_use: true` setting as a part of the input configuration.


Example configuration of using the [`container`](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-parsers-container) parser with the [`filestream`](/reference/filebeat/filebeat-input-filestream.md) input:

```yaml
filebeat.inputs:
- type: filestream
  id: unique-input-id <1>
  prospector.scanner.symlinks: true <2>
  parsers:
    - container:
        stream: stdout
        format: docker
  paths: <3>
    - '/var/log/containers/*.log'
```

1. all [`filestream`](/reference/filebeat/filebeat-input-filestream.md) inputs require a [`unique ID`](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-id).
2. container logs use symlinks, so they need to be [`enabled`](/reference/filebeat/filebeat-input-filestream.md#filebeat-input-filestream-prospector-scanner-symlinks).
3. `paths` is required.
::::

This input searches for container logs under the given path, and parse them into common message lines, extracting timestamps too. Everything happens before line filtering, multiline, and JSON decoding, so this input can be used in combination with those settings.

Example configuration:

```yaml
filebeat.inputs:
- type: container
  paths: <1>
    - '/var/log/containers/*.log'
```

1. `paths` is required. All other settings are optional.


::::{note}
*/var/log/containers/\*\*.log* is normally a symlink to */var/log/pods/\*\*/\*/.log*, so above path can be edited accordingly
::::


## Configuration options [_configuration_options_6]

The `container` input supports the following configuration options plus the [Common options](#filebeat-input-container-common-options) described later.

### `stream` [_stream]

Reads from the specified streams only: `all`, `stdout` or `stderr`. The default is `all`.


### `format` [_format]

Use the given format when reading the log file: `auto`, `docker` or `cri`. The default is `auto`, it will automatically detect the format. To disable autodetection set any of the other options.

The following input configures Filebeat to read the `stdout` stream from all containers under the default Kubernetes logs path:

```yaml
- type: container
  stream: stdout
  paths:
    - "/var/log/containers/*.log"
```


#### `encoding` [_encoding]

The file encoding to use for reading data that contains international characters. See the encoding names [recommended by the W3C for use in HTML5](http://www.w3.org/TR/encoding/).

Valid encodings:

* `plain`: plain ASCII encoding
* `utf-8` or `utf8`: UTF-8 encoding
* `gbk`: simplified Chinese charaters
* `iso8859-6e`: ISO8859-6E, Latin/Arabic
* `iso8859-6i`: ISO8859-6I, Latin/Arabic
* `iso8859-8e`: ISO8859-8E, Latin/Hebrew
* `iso8859-8i`: ISO8859-8I, Latin/Hebrew
* `iso8859-1`: ISO8859-1, Latin-1
* `iso8859-2`: ISO8859-2, Latin-2
* `iso8859-3`: ISO8859-3, Latin-3
* `iso8859-4`: ISO8859-4, Latin-4
* `iso8859-5`: ISO8859-5, Latin/Cyrillic
* `iso8859-6`: ISO8859-6, Latin/Arabic
* `iso8859-7`: ISO8859-7, Latin/Greek
* `iso8859-8`: ISO8859-8, Latin/Hebrew
* `iso8859-9`: ISO8859-9, Latin-5
* `iso8859-10`: ISO8859-10, Latin-6
* `iso8859-13`: ISO8859-13, Latin-7
* `iso8859-14`: ISO8859-14, Latin-8
* `iso8859-15`: ISO8859-15, Latin-9
* `iso8859-16`: ISO8859-16, Latin-10
* `cp437`: IBM CodePage 437
* `cp850`: IBM CodePage 850
* `cp852`: IBM CodePage 852
* `cp855`: IBM CodePage 855
* `cp858`: IBM CodePage 858
* `cp860`: IBM CodePage 860
* `cp862`: IBM CodePage 862
* `cp863`: IBM CodePage 863
* `cp865`: IBM CodePage 865
* `cp866`: IBM CodePage 866
* `ebcdic-037`: IBM CodePage 037
* `ebcdic-1040`: IBM CodePage 1140
* `ebcdic-1047`: IBM CodePage 1047
* `koi8r`: KOI8-R, Russian (Cyrillic)
* `koi8u`: KOI8-U, Ukranian (Cyrillic)
* `macintosh`: Macintosh encoding
* `macintosh-cyrillic`: Macintosh Cyrillic encoding
* `windows1250`: Windows1250, Central and Eastern European
* `windows1251`: Windows1251, Russian, Serbian (Cyrillic)
* `windows1252`: Windows1252, Legacy
* `windows1253`: Windows1253, Modern Greek
* `windows1254`: Windows1254, Turkish
* `windows1255`: Windows1255, Hebrew
* `windows1256`: Windows1256, Arabic
* `windows1257`: Windows1257, Estonian, Latvian, Lithuanian
* `windows1258`: Windows1258, Vietnamese
* `windows874`:  Windows874, ISO/IEC 8859-11, Latin/Thai
* `utf-16-bom`: UTF-16 with required BOM
* `utf-16be-bom`: big endian UTF-16 with required BOM
* `utf-16le-bom`: little endian UTF-16 with required BOM

The `plain` encoding is special, because it does not validate or transform any input.


#### `exclude_lines` [filebeat-input-container-exclude-lines]

A list of regular expressions to match the lines that you want Filebeat to exclude. Filebeat drops any lines that match a regular expression in the list. By default, no lines are dropped. Empty lines are ignored.

If [multiline](/reference/filebeat/multiline-examples.md#multiline) settings are also specified, each multiline message is combined into a single line before the lines are filtered by `exclude_lines`.

The following example configures Filebeat to drop any lines that start with `DBG`.

```yaml
filebeat.inputs:
- type: container
  ...
  exclude_lines: ['^DBG']
```

See [Regular expression support](/reference/filebeat/regexp-support.md) for a list of supported regexp patterns.


#### `include_lines` [filebeat-input-container-include-lines]

A list of regular expressions to match the lines that you want Filebeat to include. Filebeat exports only the lines that match a regular expression in the list. By default, all lines are exported. Empty lines are ignored.

If [multiline](/reference/filebeat/multiline-examples.md#multiline) settings also specified, each multiline message is combined into a single line before the lines are filtered by `include_lines`.

The following example configures Filebeat to export any lines that start with `ERR` or `WARN`:

```yaml
filebeat.inputs:
- type: container
  ...
  include_lines: ['^ERR', '^WARN']
```

::::{note}
If both `include_lines` and `exclude_lines` are defined, Filebeat executes `include_lines` first and then executes `exclude_lines`. The order in which the two options are defined doesn’t matter. The `include_lines` option will always be executed before the `exclude_lines` option, even if `exclude_lines` appears before `include_lines` in the config file.
::::


The following example exports all log lines that contain `sometext`, except for lines that begin with `DBG` (debug messages):

```yaml
filebeat.inputs:
- type: container
  ...
  include_lines: ['sometext']
  exclude_lines: ['^DBG']
```

See [Regular expression support](/reference/filebeat/regexp-support.md) for a list of supported regexp patterns.


#### `harvester_buffer_size` [_harvester_buffer_size]

The size in bytes of the buffer that each harvester uses when fetching a file. The default is 16384.


#### `max_bytes` [_max_bytes]

The maximum number of bytes that a single log message can have. All bytes after `max_bytes` are discarded and not sent. This setting is especially useful for multiline log messages, which can get large. The default is 10MB (10485760).


#### `json` [filebeat-input-container-config-json]

These options make it possible for Filebeat to decode logs structured as JSON messages. Filebeat processes the logs line by line, so the JSON decoding only works if there is one JSON object per line.

The decoding happens before line filtering and multiline. You can combine JSON decoding with filtering and multiline if you set the `message_key` option. This can be helpful in situations where the application logs are wrapped in JSON objects, as with like it happens for example with Docker.

Example configuration:

```yaml
json.keys_under_root: true
json.add_error_key: true
json.message_key: log
```

You must specify at least one of the following settings to enable JSON parsing mode:

**`keys_under_root`**
:   By default, the decoded JSON is placed under a "json" key in the output document. If you enable this setting, the keys are copied top level in the output document. The default is false.

**`overwrite_keys`**
:   If `keys_under_root` and this setting are enabled, then the values from the decoded JSON object overwrite the fields that Filebeat normally adds (type, source, offset, etc.) in case of conflicts.

**`expand_keys`**
:   If this setting is enabled, Filebeat will recursively de-dot keys in the decoded JSON, and expand them into a hierarchical object structure. For example, `{"a.b.c": 123}` would be expanded into `{"a":{"b":{"c":123}}}`. This setting should be enabled when the input is produced by an [ECS logger](https://github.com/elastic/ecs-logging).

**`add_error_key`**
:   If this setting is enabled, Filebeat adds a "error.message" and "error.type: json" key in case of JSON unmarshalling errors or when a `message_key` is defined in the configuration but cannot be used.

**`message_key`**
:   An optional configuration setting that specifies a JSON key on which to apply the line filtering and multiline settings. If specified the key must be at the top level in the JSON object and the value associated with the key must be a string, otherwise no filtering or multiline aggregation will occur.

**`document_id`**
:   Option configuration setting that specifies the JSON key to set the document id. If configured, the field will be removed from the original json document and stored in `@metadata._id`

**`ignore_decoding_error`**
:   An optional configuration setting that specifies if JSON decoding errors should be logged or not. If set to true, errors will not be logged. The default is false.


#### `multiline` [_multiline_2]

Options that control how Filebeat deals with log messages that span multiple lines. See [Multiline messages](/reference/filebeat/multiline-examples.md) for more information about configuring multiline options.


#### `exclude_files` [filebeat-input-container-exclude-files]

A list of regular expressions to match the files that you want Filebeat to ignore. By default no files are excluded.

The following example configures Filebeat to ignore all the files that have a `gz` extension:

```yaml
filebeat.inputs:
- type: container
  ...
  exclude_files: ['\.gz$']
```

See [Regular expression support](/reference/filebeat/regexp-support.md) for a list of supported regexp patterns.


#### `ignore_older` [filebeat-input-container-ignore-older]

If this option is enabled, Filebeat ignores any files that were modified before the specified timespan. Configuring `ignore_older` can be especially useful if you keep log files for a long time. For example, if you want to start Filebeat, but only want to send the newest files and files from last week, you can configure this option.

You can use time strings like 2h (2 hours) and 5m (5 minutes). The default is 0, which disables the setting. Commenting out the config has the same effect as setting it to 0.

::::{important}
You must set `ignore_older` to be greater than `close_inactive`.
::::


The files affected by this setting fall into two categories:

* Files that were never harvested
* Files that were harvested but weren’t updated for longer than `ignore_older`

For files which were never seen before, the offset state is set to the end of the file. If a state already exist, the offset is not changed. In case a file is updated again later, reading continues at the set offset position.

The `ignore_older` setting relies on the modification time of the file to determine if a file is ignored. If the modification time of the file is not updated when lines are written to a file (which can happen on Windows), the `ignore_older` setting may cause Filebeat to ignore files even though content was added at a later time.

To remove the state of previously harvested files from the registry file, use the `clean_inactive` configuration option.

Before a file can be ignored by Filebeat, the file must be closed. To ensure a file is no longer being harvested when it is ignored, you must set `ignore_older` to a longer duration than `close_inactive`.

If a file that’s currently being harvested falls under `ignore_older`, the harvester will first finish reading the file and close it after `close_inactive` is reached. Then, after that, the file will be ignored.


#### `close_*` [filebeat-input-container-close-options]

The `close_*` configuration options are used to close the harvester after a certain criteria or time. Closing the harvester means closing the file handler. If a file is updated after the harvester is closed, the file will be picked up again after `scan_frequency` has elapsed. However, if the file is moved or deleted while the harvester is closed, Filebeat will not be able to pick up the file again, and any data that the harvester hasn’t read will be lost. The `close_*` settings are applied synchronously when Filebeat attempts to read from a file, meaning that if Filebeat is in a blocked state due to blocked output, full queue or other issue, a file that would otherwise be closed remains open until Filebeat once again attempts to read from the file.


#### `close_inactive` [filebeat-input-container-close-inactive]

When this option is enabled, Filebeat closes the file handle if a file has not been harvested for the specified duration. The counter for the defined period starts when the last log line was read by the harvester. It is not based on the modification time of the file. If the closed file changes again, a new harvester is started and the latest changes will be picked up after `scan_frequency` has elapsed.

We recommended that you set `close_inactive` to a value that is larger than the least frequent updates to your log files. For example, if your log files get updated every few seconds, you can safely set `close_inactive` to `1m`. If there are log files with very different update rates, you can use multiple configurations with different values.

Setting `close_inactive` to a lower value means that file handles are closed sooner. However this has the side effect that new log lines are not sent in near real time if the harvester is closed.

The timestamp for closing a file does not depend on the modification time of the file. Instead, Filebeat uses an internal timestamp that reflects when the file was last harvested. For example, if `close_inactive` is set to 5 minutes, the countdown for the 5 minutes starts after the harvester reads the last line of the file.

You can use time strings like 2h (2 hours) and 5m (5 minutes). The default is 5m.


#### `close_renamed` [filebeat-input-container-close-renamed]

::::{warning}
Only use this option if you understand that data loss is a potential side effect.
::::


When this option is enabled, Filebeat closes the file handler when a file is renamed. This happens, for example, when rotating files. By default, the harvester stays open and keeps reading the file because the file handler does not depend on the file name. If the `close_renamed` option is enabled and the file is renamed or moved in such a way that it’s no longer matched by the file patterns specified for the path, the file will not be picked up again. Filebeat will not finish reading the file.

Do not use this option when `path` based `file_identity` is configured. It does not make sense to enable the option, as Filebeat cannot detect renames using path names as unique identifiers.

WINDOWS: If your Windows log rotation system shows errors because it can’t rotate the files, you should enable this option.


#### `close_removed` [filebeat-input-container-close-removed]

When this option is enabled, Filebeat closes the harvester when a file is removed. Normally a file should only be removed after it’s inactive for the duration specified by `close_inactive`. However, if a file is removed early and you don’t enable `close_removed`, Filebeat keeps the file open to make sure the harvester has completed. If this setting results in files that are not completely read because they are removed from disk too early, disable this option.

This option is enabled by default. If you disable this option, you must also disable `clean_removed`.

WINDOWS: If your Windows log rotation system shows errors because it can’t rotate files, make sure this option is enabled.


#### `close_eof` [filebeat-input-container-close-eof]

::::{warning}
Only use this option if you understand that data loss is a potential side effect.
::::


When this option is enabled, Filebeat closes a file as soon as the end of a file is reached. This is useful when your files are only written once and not updated from time to time. For example, this happens when you are writing every single log event to a new file. This option is disabled by default.


#### `close_timeout` [filebeat-input-container-close-timeout]

::::{warning}
Only use this option if you understand that data loss is a potential side effect. Another side effect is that multiline events might not be completely sent before the timeout expires.
::::


When this option is enabled, Filebeat gives every harvester a predefined lifetime. Regardless of where the reader is in the file, reading will stop after the `close_timeout` period has elapsed. This option can be useful for older log files when you want to spend only a predefined amount of time on the files. While `close_timeout` will close the file after the predefined timeout, if the file is still being updated, Filebeat will start a new harvester again per the defined `scan_frequency`. And the close_timeout for this harvester will start again with the countdown for the timeout.

This option is particularly useful in case the output is blocked, which makes Filebeat keep open file handlers even for files that were deleted from the disk. Setting `close_timeout` to `5m` ensures that the files are periodically closed so they can be freed up by the operating system.

If you set `close_timeout` to equal `ignore_older`, the file will not be picked up if it’s modified while the harvester is closed. This combination of settings normally leads to data loss, and the complete file is not sent.

When you use `close_timeout` for logs that contain multiline events, the harvester might stop in the middle of a multiline event, which means that only parts of the event will be sent. If the harvester is started again and the file still exists, only the second part of the event will be sent.

This option is set to 0 by default which means it is disabled.


#### `clean_*` [filebeat-input-container-clean-options]

The `clean_*` options are used to clean up the state entries in the registry file. These settings help to reduce the size of the registry file and can prevent a potential [inode reuse issue](/reference/filebeat/inode-reuse-issue.md).


#### `clean_inactive` [filebeat-input-container-clean-inactive]

::::{warning}
Only use this option if you understand that data loss is a potential side effect.
::::


When this option is enabled, Filebeat removes the state of a file after the specified period of inactivity has elapsed. The  state can only be removed if the file is already ignored by Filebeat (the file is older than `ignore_older`). The `clean_inactive` setting must be greater than `ignore_older + scan_frequency` to make sure that no states are removed while a file is still being harvested. Otherwise, the setting could result in Filebeat resending the full content constantly because  `clean_inactive` removes state for files that are still detected by Filebeat. If a file is updated or appears again, the file is read from the beginning.

The `clean_inactive` configuration option is useful to reduce the size of the registry file, especially if a large amount of new files are generated every day.

This config option is also useful to prevent Filebeat problems resulting from inode reuse on Linux. For more information, see [Inode reuse causes Filebeat to skip lines](/reference/filebeat/inode-reuse-issue.md).

::::{note}
Every time a file is renamed, the file state is updated and the counter for `clean_inactive` starts at 0 again.
::::


::::{tip}
During testing, you might notice that the registry contains state entries that should be removed based on the `clean_inactive` setting. This happens because Filebeat doesn’t remove the entries until it opens the registry again to read a different file. If you are testing the `clean_inactive` setting, make sure Filebeat is configured to read from more than one file, or the file state will never be removed from the registry.
::::



#### `clean_removed` [filebeat-input-container-clean-removed]

When this option is enabled, Filebeat cleans files from the registry if they cannot be found on disk anymore under the last known name. This means also files which were renamed after the harvester was finished will be removed. This option is enabled by default.

If a shared drive disappears for a short period and appears again, all files will be read again from the beginning because the states were removed from the registry file. In such cases, we recommend that you disable the `clean_removed` option.

You must disable this option if you also disable `close_removed`.


#### `scan_frequency` [filebeat-input-container-scan-frequency]

How often Filebeat checks for new files in the paths that are specified for harvesting. For example, if you specify a glob like `/var/log/*`, the directory is scanned for files using the frequency specified by `scan_frequency`. Specify 1s to scan the directory as frequently as possible without causing Filebeat to scan too frequently. We do not recommend to set this value `<1s`.

If you require log lines to be sent in near real time do not use a very low `scan_frequency` but adjust `close_inactive` so the file handler stays open and constantly polls your files.

The default setting is 10s.


#### `scan.sort` [filebeat-input-container-scan-sort]

::::{warning}
This functionality is in technical preview and may be changed or removed in a future release. Elastic will work to fix any issues, but features in technical preview are not subject to the support SLA of official GA features.
::::


If you specify a value other than the empty string for this setting you can determine whether to use ascending or descending order using `scan.order`. Possible values are `modtime` and `filename`. To sort by file modification time, use `modtime`, otherwise use `filename`. Leave this option empty to disable it.

If you specify a value for this setting, you can use `scan.order` to configure whether files are scanned in ascending or descending order.

The default setting is disabled.


#### `scan.order` [filebeat-input-container-scan-order]

::::{warning}
This functionality is in technical preview and may be changed or removed in a future release. Elastic will work to fix any issues, but features in technical preview are not subject to the support SLA of official GA features.
::::


Specifies whether to use ascending or descending order when `scan.sort` is set to a value other than none. Possible values are `asc` or `desc`.

The default setting is `asc`.


#### `tail_files` [_tail_files]

If this option is set to true, Filebeat starts reading new files at the end of each file instead of the beginning. When this option is used in combination with log rotation, it’s possible that the first log entries in a new file might be skipped. The default setting is false.

This option applies to files that Filebeat has not already processed. If you ran Filebeat previously and the state of the file was already persisted, `tail_files` will not apply. Harvesting will continue at the previous offset. To apply `tail_files` to all files, you must stop Filebeat and remove the registry file. Be aware that doing this removes ALL previous states.

::::{note}
You can use this setting to avoid indexing old log lines when you run Filebeat on a set of log files for the first time. After the first run, we recommend disabling this option, or you risk losing lines during file rotation.
::::



#### `symlinks` [_symlinks]

The `symlinks` option allows Filebeat to harvest symlinks in addition to regular files. When harvesting symlinks, Filebeat opens and reads the original file even though it reports the path of the symlink.

When you configure a symlink for harvesting, make sure the original path is excluded. If a single input is configured to harvest both the symlink and the original file, Filebeat will detect the problem and only process the first file it finds. However, if two different inputs are configured (one to read the symlink and the other the original path), both paths will be harvested, causing Filebeat to send duplicate data and the inputs to overwrite each other’s state.

The `symlinks` option can be useful if symlinks to the log files have additional metadata in the file name, and you want to process the metadata in Logstash. This is, for example, the case for Kubernetes log files.

Because this option may lead to data loss, it is disabled by default.


#### `backoff` [_backoff]

The backoff options specify how aggressively Filebeat crawls open files for updates. You can use the default values in most cases.

The `backoff` option defines how long Filebeat waits before checking a file again after EOF is reached. The default is 1s, which means the file is checked every second if new lines were added. This enables near real-time crawling. Every time a new line appears in the file, the `backoff` value is reset to the initial value. The default is 1s.


#### `max_backoff` [_max_backoff]

The maximum time for Filebeat to wait before checking a file again after EOF is reached. After having backed off multiple times from checking the file, the wait time will never exceed `max_backoff` regardless of what is specified for  `backoff_factor`. Because it takes a maximum of 10s to read a new line, specifying 10s for `max_backoff` means that, at the worst, a new line could be added to the log file if Filebeat has backed off multiple times. The default is 10s.

Requirement: Set `max_backoff` to be greater than or equal to `backoff` and less than or equal to `scan_frequency` (`backoff <= max_backoff <= scan_frequency`). If `max_backoff` needs to be higher, it is recommended to close the file handler instead and let Filebeat pick up the file again.


#### `backoff_factor` [_backoff_factor]

This option specifies how fast the waiting time is increased. The bigger the backoff factor, the faster the `max_backoff` value is reached. The backoff factor increments exponentially. The minimum value allowed is 1. If this value is set to 1, the backoff algorithm is disabled, and the `backoff` value is used for waiting for new lines. The `backoff` value will be multiplied each time with the `backoff_factor` until `max_backoff` is reached. The default is 2.


#### `harvester_limit` [filebeat-input-container-harvester-limit]

The `harvester_limit` option limits the number of harvesters that are started in parallel for one input. This directly relates to the maximum number of file handlers that are opened. The default for `harvester_limit` is 0, which means there is no limit. This configuration is useful if the number of files to be harvested exceeds the open file handler limit of the operating system.

Setting a limit on the number of harvesters means that potentially not all files are opened in parallel. Therefore we recommended that you use this option in combination with the `close_*` options to make sure harvesters are stopped more often so that new files can be picked up.

Currently if a new harvester can be started again, the harvester is picked randomly. This means it’s possible that the harvester for a file that was just closed and then updated again might be started instead of the harvester for a file that hasn’t been harvested for a longer period of time.

This configuration option applies per input. You can use this option to indirectly set higher priorities on certain inputs by assigning a higher limit of harvesters.


#### `file_identity` [_file_identity]

Different `file_identity` methods can be configured to suit the environment where you are collecting log messages.

**`native`**
:   The default behaviour of Filebeat is to differentiate between files using their inodes and device ids.

```yaml
file_identity.native: ~
```

**`path`**
:   To identify files based on their paths use this strategy.

::::{warning}
Only use this strategy if your log files are rotated to a folder outside of the scope of your input or not at all. Otherwise you end up with duplicated events.
::::


::::{warning}
This strategy does not support renaming files. If an input file is renamed, Filebeat will read it again if the new path matches the settings of the input.
::::


```yaml
file_identity.path: ~
```

**`inode_marker`**
:   If the device id changes from time to time, you must use this method to distinguish files. This option is not supported on Windows.

Set the location of the marker file the following way:

```yaml
file_identity.inode_marker.path: /logs/.filebeat-marker
```



## Common options [filebeat-input-container-common-options]

The following configuration options are supported by all inputs.


#### `enabled` [_enabled_6]

Use the `enabled` option to enable and disable inputs. By default, enabled is set to true.


#### `tags` [_tags_6]

A list of tags that Filebeat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
filebeat.inputs:
- type: container
  . . .
  tags: ["json"]
```


#### `fields` [filebeat-input-container-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
filebeat.inputs:
- type: container
  . . .
  fields:
    app_id: query_engine_12
```


#### `fields_under_root` [fields-under-root-container]

If this option is set to true, the custom [fields](#filebeat-input-container-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


#### `processors` [_processors_6]

A list of processors to apply to the input data.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


#### `pipeline` [_pipeline_6]

The ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::


::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::



#### `keep_null` [_keep_null_6]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `index` [_index_6]

If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.11.01"`.


#### `publisher_pipeline.disable_host` [_publisher_pipeline_disable_host_6]

By default, all events contain `host.name`. This option can be set to `true` to disable the addition of this field to all events. The default value is `false`.
