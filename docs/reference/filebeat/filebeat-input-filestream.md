---
navigation_title: "filestream"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-input-filestream.html
---

# filestream input [filebeat-input-filestream]


Use the `filestream` input to read lines from active log files. It is the new, improved alternative to the `log` input. It comes with various improvements to the existing input:

1. Checking of `close.on_state_change.*` options happens out of band. Thus, if an output is blocked, Filebeat can close the reader and avoid keeping too many files open.
2. Detailed metrics are available for all files that match the `paths` configuration regardless of the `harvester_limit`. This way, you can keep track of all files, even ones that are not actively read.
3. The order of `parsers` is configurable. So it is possible to parse JSON lines and then aggregate the contents into a multiline event.
4. Some position updates and metadata changes no longer depend on the publishing pipeline. If the pipeline is blocked some changes are still applied to the registry.
5. Only the most recent updates are serialized to the registry. In contrast, the `log` input has to serialize the complete registry on each ACK from the outputs. This makes the registry updates much quicker with this input.
6. The input ensures that only offsets updates are written to the registry append only log. The `log` writes the complete file state.
7. Stale entries can be removed from the registry, even if there is no active input.
8. The default behaviour is to identify files based on their contents using the [`fingerprint`](#filebeat-input-filestream-file-identity-fingerprint) [`file_identity`](#filebeat-input-filestream-file-identity) This solves data duplication caused by inode reuse.

To configure this input, specify a list of glob-based [`paths`](#filestream-input-paths) that must be crawled to locate and fetch the log lines.

Example configuration:

```yaml
filebeat.inputs:
- type: filestream
  id: my-filestream-id
  paths:
    - /var/log/messages
    - /var/log/*.log
```

::::{warning}
Each filestream input must have a unique ID. Omitting or changing the filestream ID may cause data duplication. Without a unique ID, filestream is unable to correctly track the state of files.
::::


You can apply additional [configuration settings](#filebeat-input-filestream-options) (such as `fields`, `include_lines`, `exclude_lines` and so on) to the lines harvested from these files. The options that you specify are applied to all the files harvested by this input.

To apply different configuration settings to different files, you need to define multiple input sections:

```yaml
filebeat.inputs:
- type: filestream <1>
  id: my-filestream-id
  paths:
    - /var/log/system.log
    - /var/log/wifi.log
- type: filestream <2>
  id: apache-filestream-id
  paths:
    - "/var/log/apache2/*"
  fields:
    apache: true
```

1. Harvests lines from two files:  `system.log` and `wifi.log`.
2. Harvests lines from every file in the `apache2` directory, and uses the `fields` configuration option to add a field called `apache` to the output.


## Reading files on network shares and cloud providers [filestream-file-identity]

::::{warning}
Some file identity methods do not support reading from network shares and cloud providers, to avoid duplicating events, use the default `file_identity`: `fingerprint`.
::::


::::{important}
Changing `file_identity` is only supported when migrating from `native` or `path` to `fingerprint`.
::::


::::{warning}
Any unsupported change in `file_identity` methods between runs may result in duplicated events in the output.
::::


`fingerprint` is the default and recommended file identity because it does not rely on the file system/OS, it generates a hash from a portion of the file (the first 1024 bytes, by default) and uses that to identify the file. This works well with log rotation strategies that move/rename the file and on Windows as file identifiers might be more volatile. The downside is that Filebeat will wait until the file reaches 1024 bytes before start ingesting any file.

::::{warning}
Once this file identity is enabled, changing the fingerprint configuration (offset, length, etc) will lead to a global re-ingestion of all files that match the paths configuration of the input.
::::


Please refer to the [fingerprint configuration for details](#filebeat-input-filestream-scan-fingerprint).

Selecting `path` instructs Filebeat to identify files based on their paths. This is a quick way to avoid rereading files if inode and device ids might change. However, keep in mind if the files are rotated (renamed), they will be reread and resubmitted.

The option `inode_marker` can be used if the inodes stay the same even if the device id is changed. You should choose this method if your files are rotated instead of `path` if possible. You have to configure a marker file readable by Filebeat and set the path in the option `path` of `inode_marker`.

The content of this file must be unique to the device. You can put the UUID of the device or mountpoint where the input is stored. The following example oneliner generates a hidden marker file for the selected mountpoint `/logs`: Please note that you should not use this option on Windows as file identifiers might be more volatile.

```sh
$ lsblk -o MOUNTPOINT,UUID | grep /logs | awk '{print $2}' >> /logs/.filebeat-marker
```

To set the generated file as a marker for `file_identity` you should configure the input the following way:

```yaml
filebeat.inputs:
- type: filestream
  id: my-filestream-id
  paths:
    - /logs/*.log
  file_identity.inode_marker.path: /logs/.filebeat-marker
```


## Reading from rotating logs [filestream-rotating-logs]

When dealing with file rotation, avoid harvesting symlinks. Instead use the [`paths`](#filestream-input-paths) setting to point to the original file, and specify a pattern that matches the file you want to harvest and all of its rotated files. Also make sure your log rotation strategy prevents lost or duplicate messages. For more information, see [Log rotation results in lost or duplicate events](/reference/filebeat/file-log-rotation.md).

Furthermore, to avoid duplicate of rotated log messages, do not use the `path` method for `file_identity`. Or exclude the rotated files with `exclude_files` option.


## Prospector options [filebeat-input-filestream-options]

The prospector is running a file system watcher which looks for files specified in the `paths` option. At the moment only simple file system scanning is supported.


#### `id` [filebeat-input-filestream-id]

A unique identifier for this filestream input. Each filestream input must have a unique ID. Filestream will not start inputs with duplicated IDs.

::::{warning}
Changing input ID may cause data duplication because the state of the files will be lost and they will be read from the beginning again.
::::



#### `allow_deprecated_id_duplication` [filestream-input-allow_deprecated_id_duplication]

This allows Filebeat to run multiple instances of the filestream input with the same ID. This is intended to add backwards compatibility with the behaviour prior to 9.0. It defaults to `false` and is **not recommended** in new configurations.

This setting is per input, so make sure to enable it in all filestream inputs that use duplicated IDs.

::::{warning}
Duplicated IDs will lead to data duplication and some input instances will not produce any metrics.
::::



#### `paths` [filestream-input-paths]

A list of glob-based paths that will be crawled and fetched. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, to fetch all files from a predefined level of subdirectories, the following pattern can be used: `/var/log/*/*.log`. This fetches all `.log` files from the subfolders of `/var/log`. It does not fetch log files from the `/var/log` folder itself. It is possible to recursively fetch all files in all subdirectories of a directory using the optional [`recursive_glob`](#filestream-recursive-glob) settings.

Filebeat starts a harvester for each file that it finds under the specified paths. You can specify one path per line. Each line begins with a dash (-).


## Scanner options [_scanner_options]

The scanner watches the configured paths. It scans the file system periodically and returns the file system events to the Prospector.


#### `prospector.scanner.recursive_glob` [filestream-recursive-glob]

Enable expanding `**` into recursive glob patterns. With this feature enabled, the rightmost `**` in each path is expanded into a fixed number of glob patterns. For example: `/foo/**` expands to `/foo`, `/foo/*`, `/foo/*/*`, and so on. If enabled it expands a single `**` into a 8-level deep `*` pattern.

This feature is enabled by default. Set `prospector.scanner.recursive_glob` to false to disable it.


#### `prospector.scanner.exclude_files` [filebeat-input-filestream-exclude-files]

A list of regular expressions to match the files that you want Filebeat to ignore. By default no files are excluded.

The following example configures Filebeat to ignore all the files that have a `gz` extension:

```yaml
filebeat.inputs:
- type: filestream
  ...
  prospector.scanner.exclude_files: ['\.gz$']
```

See [Regular expression support](/reference/filebeat/regexp-support.md) for a list of supported regexp patterns.

### `prospector.scanner.include_files` [_prospector_scanner_include_files]

A list of regular expressions to match the files that you want Filebeat to include. If a list of regexes is provided, only the files that are allowed by the patterns are harvested.

By default no files are excluded. This option is the counterpart of `prospector.scanner.exclude_files`.

The following example configures Filebeat to exclude files that are not under `/var/log`:

```yaml
filebeat.inputs:
- type: filestream
  ...
  prospector.scanner.include_files: ['^/var/log/.*']
```

::::{note}
Patterns should start with `^` in case of absolute paths.
::::


See [Regular expression support](/reference/filebeat/regexp-support.md) for a list of supported regexp patterns.


### `prospector.scanner.symlinks` [filebeat-input-filestream-prospector-scanner-symlinks]

The `symlinks` option allows Filebeat to harvest symlinks in addition to regular files. When harvesting symlinks, Filebeat opens and reads the original file even though it reports the path of the symlink.

When you configure a symlink for harvesting, make sure the original path is excluded. If a single input is configured to harvest both the symlink and the original file, Filebeat will detect the problem and only process the first file it finds. However, if two different inputs are configured (one to read the symlink and the other the original path), both paths will be harvested, causing Filebeat to send duplicate data and the inputs to overwrite each other’s state.

The `symlinks` option can be useful if symlinks to the log files have additional metadata in the file name, and you want to process the metadata in Logstash. This is, for example, the case for Kubernetes log files.

Because this option may lead to data loss, it is disabled by default.


### `prospector.scanner.resend_on_touch` [_prospector_scanner_resend_on_touch]

If this option is enabled a file is resent if its size has not changed but its modification time has changed to a later time than before. It is disabled by default to avoid accidentally resending files.


#### `prospector.scanner.check_interval` [filebeat-input-filestream-scan-frequency]

How often Filebeat checks for new files in the paths that are specified for harvesting. For example, if you specify a glob like `/var/log/*`, the directory is scanned for files using the frequency specified by `check_interval`. Specify 1s to scan the directory as frequently as possible without causing Filebeat to scan too frequently. We do not recommend to set this value `<1s`.

If you require log lines to be sent in near real time do not use a very low `check_interval` but adjust `close.on_state_change.inactive` so the file handler stays open and constantly polls your files.

The default setting is 10s.


#### `prospector.scanner.fingerprint` [filebeat-input-filestream-scan-fingerprint]

Instead of relying on the device ID and inode values when comparing files, compare hashes of the given byte ranges of files. This is the default behaviour for Filebeat.

Following are some scenarios where this can happen:

1. Some file systems (i.e. in Docker) cache and re-use inodes

    for example if you:

    1. Create a file (`touch x`)
    2. Check the file’s inode (`ls -i x`)
    3. Delete the file (`rm x`)
    4. Create a new file right away (`touch y`)
    5. Check the inode of the new file (`ls -i y`)

        For both files you might see the same inode value despite even having different filenames.

2. Non-Ext file systems can change inodes:

    Ext file systems store the inode number in the `i_ino` file, inside a struct `inode`, which is written to disk. In this case, if the file is the same (not another file with the same name) then the inode number is guaranteed to be the same.

    If the file system is other than Ext, the inode number is generated by the inode operations defined by the file system driver. As they don’t have the concept of what an inode is, they have to mimic all of the inode’s internal fields to comply with VFS, so this number will probably be different after a reboot, even after closing and opening the file again (theoretically).

3. Some file processing tools change inode values

    Sometimes users unintentionally change inodes by using tools like `rsync` or `sed`.

4. Some operating systems change device IDs after reboot

    Depending on a mounting approach, the device ID (which is also used for comparing files) might change after a reboot.


**Configuration**

Fingerprint mode is disabled by default.

::::{warning}
Enabling fingerprint mode delays ingesting new files until they grow to at least `offset`+`length` bytes in size, so they can be fingerprinted. Until then these files are ignored.
::::


Normally, log lines contain timestamps and other unique fields that should be able to use the fingerprint mode, but in every use-case users should inspect their logs to determine what are the appropriate values for the `offset` and `length` parameters. Default `offset` is `0` and default `length` is `1024` or 1 KB. `length` cannot be less than `64`.

```yaml
fingerprint:
  enabled: false
  offset: 0
  length: 1024
```


#### `ignore_older` [filebeat-input-filestream-ignore-older]

If this option is enabled, Filebeat ignores any files that were modified before the specified timespan. Configuring `ignore_older` can be especially useful if you keep log files for a long time. For example, if you want to start Filebeat, but only want to send the newest files and files from last week, you can configure this option.

You can use time strings like 2h (2 hours) and 5m (5 minutes). The default is 0, which disables the setting. Commenting out the config has the same effect as setting it to 0.

::::{important}
You must set `ignore_older` to be greater than `close.on_state_change.inactive`.
::::


The files affected by this setting fall into two categories:

* Files that were never harvested
* Files that were harvested but weren’t updated for longer than `ignore_older`

For files which were never seen before, the offset state is set to the end of the file. If a state already exists, the offset is reset to the size of the file. If a file is updated again later, reading continues at the set offset position.

The `ignore_older` setting relies on the modification time of the file to determine if a file is ignored. If the modification time of the file is not updated when lines are written to a file (which can happen on Windows), the `ignore_older` setting may cause Filebeat to ignore files even though content was added at a later time.

To remove the state of previously harvested files from the registry file, use the `clean_inactive` configuration option.

Before a file can be ignored by Filebeat, the file must be closed. To ensure a file is no longer being harvested when it is ignored, you must set `ignore_older` to a longer duration than `close.on_state_change.inactive`.

If a file that’s currently being harvested falls under `ignore_older`, the harvester will first finish reading the file and close it after `close.on_state_change.inactive` is reached. Then, after that, the file will be ignored.


#### `ignore_inactive` [filebeat-input-filestream-ignore-inactive]

If this option is enabled, Filebeat ignores every file that has not been updated since the selected time. Possible options are `since_first_start` and `since_last_start`. The first option ignores every file that has not been updated since the first start of Filebeat. It is useful when the Beat might be restarted due to configuration changes or a failure. The second option tells the Beat to read from files that have been updated since its start.

The files affected by this setting fall into two categories:

* Files that were never harvested
* Files that were harvested but weren’t updated since `ignore_inactive`.

For files that were never seen before, the offset state is set to the end of the file. If a state already exist, the offset is not changed. In case a file is updated again later, reading continues at the set offset position.

The setting relies on the modification time of the file to determine if a file is ignored. If the modification time of the file is not updated when lines are written to a file (which can happen on Windows), the setting may cause Filebeat to ignore files even though content was added at a later time.

To remove the state of previously harvested files from the registry file, use the `clean_inactive` configuration option.

## Take over [filebeat-input-filestream-take-over]
When `take_over` is enabled, this `filestream` input will take over
states from the [`log`](/reference/filebeat/filebeat-input-log.md) input
or other `filestream` inputs. Only states of files being actively
harvested by this input are taken over.

To take over files from a `log` input, simply set `take_over.enabled: true`.

To take over states from other `filestream` inputs, set
`take_over.enabled: true` and set `take_over.from_ids` to a list of
existing `filestream` IDs you want to migrate files from.

On both cases make sure the files you want this input to take over
match the configured globs in `paths`.

When `take_over.from_ids` is set, files are not taken over from `log`
inputs. The migration is limited to `filestream` inputs only.

```yaml
take_over:
  enabled: true
  from_ids: ["foo", "bar"] # omit to take over from the log input
```
:::{important}
The `take over` mode can work correctly only if the source (taken from) inputs are no longer active. If source inputs are still harvesting the files which are being migrated, it will lead to data duplication and in some cases might cause data loss.
:::

::::{important}
`take_over.enabled: true` requires the `filestream` to have a unique ID.
::::


This `take over` mode was created to enable smooth migration from
deprecated `log` inputs to the new `filestream` inputs and to allow
changing `filestream` input IDs without data re-ingestion.

See [*Migrate `log` input configurations to `filestream`*](/reference/filebeat/migrate-to-filestream.md) for more details about the migration process.

::::{warning}
The `take over` mode is still in beta, however, it should be generally safe to use.
::::


### Limitations
Take over can only migrate states from existing files that are not
ignored during the `filestream` input start up. Once the input is
ingesting data, if a new file appears, `filestream` will not try to
migrate its state.

#### `close.*` [filebeat-input-filestream-close-options]

The `close.*` configuration options are used to close the harvester after a certain criteria or time. Closing the harvester means closing the file handler. If a file is updated after the harvester is closed, the file will be picked up again after `prospector.scanner.check_interval` has elapsed. However, if the file is moved or deleted while the harvester is closed, Filebeat will not be able to pick up the file again, and any data that the harvester hasn’t read will be lost.

The `close.on_state_change.*` settings are applied asynchronously to read from a file, meaning that if Filebeat is in a blocked state due to blocked output, full queue or other issue, a file that would be closed regardless.


#### `close.on_state_change.inactive` [filebeat-input-filestream-close-inactive]

When this option is enabled, Filebeat closes the file handle if a file has not been harvested for the specified duration. The counter for the defined period starts when the last log line was read by the harvester. It is not based on the modification time of the file. If the closed file changes again, a new harvester is started and the latest changes will be picked up after `prospector.scanner.check_interval` has elapsed.

We recommended that you set `close.on_state_change.inactive` to a value that is larger than the least frequent updates to your log files. For example, if your log files get updated every few seconds, you can safely set `close.on_state_change.inactive` to `1m`. If there are log files with very different update rates, you can use multiple configurations with different values.

Setting `close.on_state_change.inactive` to a lower value means that file handles are closed sooner. However this has the side effect that new log lines are not sent in near real time if the harvester is closed.

The timestamp for closing a file does not depend on the modification time of the file. Instead, Filebeat uses an internal timestamp that reflects when the file was last harvested. For example, if `close.on_state_change.inactive` is set to 5 minutes, the countdown for the 5 minutes starts after the harvester reads the last line of the file.

You can use time strings like 2h (2 hours) and 5m (5 minutes). The default is 5m.


#### `close.on_state_change.renamed` [filebeat-input-filestream-close-renamed]

::::{warning}
Only use this option if you understand that data loss is a potential side effect.
::::


When this option is enabled, Filebeat closes the file handler when a file is renamed. This happens, for example, when rotating files. By default, the harvester stays open and keeps reading the file because the file handler does not depend on the file name. If the `close.on_state_change.renamed` option is enabled and the file is renamed or moved in such a way that it’s no longer matched by the file patterns specified for the , the file will not be picked up again. Filebeat will not finish reading the file.

Do not use this option when `path` based `file_identity` is configured. It does not make sense to enable the option, as Filebeat cannot detect renames using path names as unique identifiers.

WINDOWS: If your Windows log rotation system shows errors because it can’t rotate the files, you should enable this option.


#### `close.on_state_change.removed` [filebeat-input-filestream-close-removed]

When this option is enabled, Filebeat closes the harvester when a file is removed. Normally a file should only be removed after it’s inactive for the duration specified by `close.on_state_change.inactive`. However, if a file is removed early and you don’t enable `close.on_state_change.removed`, Filebeat keeps the file open to make sure the harvester has completed. If this setting results in files that are not completely read because they are removed from disk too early, disable this option.

This option is enabled by default. If you disable this option, you must also disable `clean_removed`.

WINDOWS: If your Windows log rotation system shows errors because it can’t rotate files, make sure this option is enabled.


#### `close.reader.on_eof` [filebeat-input-filestream-close-eof]

::::{warning}
Only use this option if you understand that data loss is a potential side effect.
::::


When this option is enabled, Filebeat closes a file as soon as the end of a file is reached. This is useful when your files are only written once and not updated from time to time. For example, this happens when you are writing every single log event to a new file. This option is disabled by default.


#### `close.reader.after_interval` [filebeat-input-filestream-close-timeout]

::::{warning}
Only use this option if you understand that data loss is a potential side effect. Another side effect is that multiline events might not be completely sent before the timeout expires.
::::


When this option is enabled, Filebeat gives every harvester a predefined lifetime. Regardless of where the reader is in the file, reading will stop after the `close.reader.after_interval` period has elapsed. This option can be useful for older log files when you want to spend only a predefined amount of time on the files. While `close.reader.after_interval` will close the file after the predefined timeout, if the file is still being updated, Filebeat will start a new harvester again per the defined `prospector.scanner.check_interval`. And the close.reader.after_interval for this harvester will start again with the countdown for the timeout.

This option is particularly useful in case the output is blocked, which makes Filebeat keep open file handlers even for files that were deleted from the disk. Setting `close.reader.after_interval` to `5m` ensures that the files are periodically closed so they can be freed up by the operating system.

If you set `close.reader.after_interval` to equal `ignore_older`, the file will not be picked up if it’s modified while the harvester is closed. This combination of settings normally leads to data loss, and the complete file is not sent.

When you use `close.reader.after_interval` for logs that contain multiline events, the harvester might stop in the middle of a multiline event, which means that only parts of the event will be sent. If the harvester is started again and the file still exists, only the second part of the event will be sent.

This option is set to 0 by default which means it is disabled.


#### `clean_*` [filebeat-input-filestream-clean-options]

The `clean_*` options are used to clean up the state entries in the registry file. These settings help to reduce the size of the registry file and can prevent a potential [inode reuse issue](/reference/filebeat/inode-reuse-issue.md).


#### `clean_inactive` [filebeat-input-filestream-clean-inactive]

::::{warning}
Only use this option if you understand that data loss is a potential side effect.
::::


When this option is enabled, Filebeat removes the state of a file after the specified period of inactivity has elapsed. The state can only be removed if the file is already ignored by Filebeat (the file is older than `ignore_older`). The `clean_inactive` setting must be greater than `ignore_older + prospector.scanner.check_interval` to make sure that no states are removed while a file is still being harvested. Otherwise, the setting could result in Filebeat resending the full content constantly because `clean_inactive` removes state for files that are still detected by Filebeat. If a file is updated or appears again, the file is read from the beginning.

The `clean_inactive` configuration option is useful to reduce the size of the registry file, especially if a large amount of new files are generated every day.

This config option is also useful to prevent Filebeat problems resulting from inode reuse on Linux. For more information, see [Inode reuse causes Filebeat to skip lines](/reference/filebeat/inode-reuse-issue.md).

::::{note}
Every time a file is renamed, the file state is updated and the counter for `clean_inactive` starts at 0 again.
::::


::::{tip}
During testing, you might notice that the registry contains state entries that should be removed based on the `clean_inactive` setting. This happens because Filebeat doesn’t remove the entries until the registry garbage collector (GC) runs. Once the TTL for a state expired, there are no active harvesters for the file and the registry GC runs, then, and only then the state is removed from memory and an `op: remove` is added to the registry log file.
::::



#### `clean_removed` [filebeat-input-filestream-clean-removed]

When this option is enabled, Filebeat cleans files from the registry if they cannot be found on disk anymore under the last known name. This means also files which were renamed after the harvester was finished will be removed. This option is enabled by default.

If a shared drive disappears for a short period and appears again, all files will be read again from the beginning because the states were removed from the registry file. In such cases, we recommend that you disable the `clean_removed` option.

You must disable this option if you also disable `close.on_state_change.removed`.


#### `backoff.*` [_backoff_2]

The backoff options specify how aggressively Filebeat crawls open files for updates. You can use the default values in most cases.


#### `backoff.init` [_backoff_init]

The `backoff.init` option defines how long Filebeat waits for the first time before checking a file again after EOF is reached. The backoff intervals increase exponentially. The default is 2s. Thus, the file is checked after 2 seconds, then 4 seconds, then 8 seconds and so on until it reaches the limit defined in `backoff.max`. Every time a new line appears in the file, the `backoff.init` value is reset to the initial value.


#### `backoff.max` [_backoff_max]

The maximum time for Filebeat to wait before checking a file again after EOF is reached. After having backed off multiple times from checking the file, the wait time will never exceed `backoff.max`. Because it takes a maximum of 10s to read a new line, specifying 10s for `backoff.max` means that, at the worst, a new line could be added to the log file if Filebeat has backed off multiple times. The default is 10s.

Requirement: Set `backoff.max` to be greater than or equal to `backoff.init` and less than or equal to `prospector.scanner.check_interval` (`backoff.init <= backoff.max <= prospector.scanner.check_interval`). If `backoff.max` needs to be higher, it is recommended to close the file handler instead and let Filebeat pick up the file again.


#### `harvester_limit` [filebeat-input-filestream-harvester-limit]

The `harvester_limit` option limits the number of harvesters that are started in parallel for one input. This directly relates to the maximum number of file handlers that are opened. The default for `harvester_limit` is 0, which means there is no limit. This configuration is useful if the number of files to be harvested exceeds the open file handler limit of the operating system.

Setting a limit on the number of harvesters means that potentially not all files are opened in parallel. Therefore we recommended that you use this option in combination with the `close.on_state_change.*` options to make sure harvesters are stopped more often so that new files can be picked up.

Currently if a new harvester can be started again, the harvester is picked randomly. This means it’s possible that the harvester for a file that was just closed and then updated again might be started instead of the harvester for a file that hasn’t been harvested for a longer period of time.

This configuration option applies per input. You can use this option to indirectly set higher priorities on certain inputs by assigning a higher limit of harvesters.


#### `file_identity` [filebeat-input-filestream-file-identity]

Different `file_identity` methods can be configured to suit the environment where you are collecting log messages.

Follow [this comprehensive guide](/reference/filebeat/file-identity.md) on how to choose a file identity option right for your use-case.

::::{important}
Changing `file_identity` is only supported from `native` or `path` to `fingerprint`. On those cases Filebeat will automatically migrate the state of the file when filestream starts.
::::


::::{warning}
Any unsupported change in `file_identity` methods between runs may result in duplicated events in the output.
::::


$$$filebeat-input-filestream-file-identity-fingerprint$$$

**`fingerprint`**
:   The default behaviour of Filebeat is to identify files based on content by hashing a specific range (0 to 1024 bytes by default).

::::{warning}
In order to use this file identity option, you must enable the [fingerprint option in the scanner](#filebeat-input-filestream-scan-fingerprint). Once this file identity is enabled, changing the fingerprint configuration (offset, length, or other settings) will lead to a global re-ingestion of all files that match the paths configuration of the input.
::::


Please refer to the [fingerprint configuration for details](#filebeat-input-filestream-scan-fingerprint).

```yaml
file_identity.fingerprint: ~
```

**`native`**
:   Differentiates between files using their inodes and device ids.

    In some cases these values can change during the lifetime of a file. For example, when using the Linux [LVM](https://en.wikipedia.org/wiki/Logical_Volume_Manager_%28Linux%29) (Logical Volume Manager), device numbers are allocated dynamically at module load (refer to [Persistent Device Numbers](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/7/html/logical_volume_manager_administration/lv#persistent_numbers) in the Red Hat Enterprise Linux documentation). To avoid the possibility of data duplication in this case, you can set `file_identity` to `fingerprint` rather than the default `native`.

    The states of files generated by `native` file identity can be migrated to `fingerprint`.


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


    The states of files generated by `path` file identity can be migrated to `fingerprint`.


```yaml
file_identity.path: ~
```

**`inode_marker`**
:   If the device id changes from time to time, you must use this method to distinguish files. This option is not supported on Windows.

    Set the location of the marker file the following way:


```yaml
file_identity.inode_marker.path: /logs/.filebeat-marker
```


## Log rotation [filestream-log-rotation-support]

As log files are constantly written, they must be rotated and purged to prevent the logger application from filling up the disk. Rotation is done by an external application, thus, Filebeat needs information how to cooperate with it.

When reading from rotating files make sure the paths configuration includes both the active file and all rotated files.

By default, Filebeat is able to track files correctly in the following strategies:

* create: new active file with a unique name is created on rotation
* rename: rotated files are renamed

However, in case of copytruncate strategy, you should provide additional configuration to Filebeat.


### rotation.external.strategy.copytruncate [_rotation_external_strategy_copytruncate]

::::{warning}
This functionality is in technical preview and may be changed or removed in a future release. Elastic will work to fix any issues, but features in technical preview are not subject to the support SLA of official GA features.
::::


If the log rotating application copies the contents of the active file and then truncates the original file, use these options to help Filebeat to read files correctly.

Set the option `suffix_regex` so Filebeat can tell active and rotated files apart. There are two supported suffix types in the input: numberic and date.



## Numeric suffix [_numeric_suffix]

If your rotated files have an incrementing index appended to the end of the filename, e.g. active file `apache.log` and the rotated files are named `apache.log.1`, `apache.log.2`, etc, use the following configuration.

```yaml
---
rotation.external.strategy.copytruncate:
  suffix_regex: \.\d$
---
```


## Date suffix [_date_suffix]

If the rotation date is appended to the end of the filename, e.g. active file `apache.log` and the rotated files are named `apache.log-20210526`, `apache.log-20210527`, etc. use the following configuration:

```yaml
---
rotation.external.strategy.copytruncate:
  suffix_regex: \-\d{6}$
  dateformat: -20060102
---
```


#### `encoding` [_encoding_2]

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


#### `exclude_lines` [filebeat-input-filestream-exclude-lines]

A list of regular expressions to match the lines that you want Filebeat to exclude. Filebeat drops any lines that match a regular expression in the list. By default, no lines are dropped. Empty lines are ignored.

The following example configures Filebeat to drop any lines that start with `DBG`.

```yaml
filebeat.inputs:
- type: filestream
  ...
  exclude_lines: ['^DBG']
```

See [Regular expression support](/reference/filebeat/regexp-support.md) for a list of supported regexp patterns.


#### `include_lines` [filebeat-input-filestream-include-lines]

A list of regular expressions to match the lines that you want Filebeat to include. Filebeat exports only the lines that match a regular expression in the list. By default, all lines are exported. Empty lines are ignored.

The following example configures Filebeat to export any lines that start with `ERR` or `WARN`:

```yaml
filebeat.inputs:
- type: filestream
  ...
  include_lines: ['^ERR', '^WARN']
```

::::{note}
If both `include_lines` and `exclude_lines` are defined, Filebeat executes `include_lines` first and then executes `exclude_lines`. The order in which the two options are defined doesn’t matter. The `include_lines` option will always be executed before the `exclude_lines` option, even if `exclude_lines` appears before `include_lines` in the config file.
::::


The following example exports all log lines that contain `sometext`, except for lines that begin with `DBG` (debug messages):

```yaml
filebeat.inputs:
- type: filestream
  ...
  include_lines: ['sometext']
  exclude_lines: ['^DBG']
```

See [Regular expression support](/reference/filebeat/regexp-support.md) for a list of supported regexp patterns.


#### `buffer_size` [_buffer_size]

The size in bytes of the buffer that each harvester uses when fetching a file. The default is 16384.


#### `message_max_bytes` [_message_max_bytes]

The maximum number of bytes that a single log message can have. All bytes after `message_max_bytes` are discarded and not sent. The default is 10MB (10485760).


#### `parsers` [_parsers]

This option expects a list of parsers that the log line has to go through.

Available parsers:

* `multiline`
* `ndjson`
* `container`
* `syslog`
* `include_message`

In this example, Filebeat is reading multiline messages that consist of 3 lines and are encapsulated in single-line JSON objects. The multiline message is stored under the key `msg`.

```yaml
filebeat.inputs:
- type: filestream
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


#### `multiline` [_multiline_3]

Options that control how Filebeat deals with log messages that span multiple lines. See [Multiline messages](/reference/filebeat/multiline-examples.md) for more information about configuring multiline options.


#### `ndjson` [filebeat-input-filestream-ndjson]

These options make it possible for Filebeat to decode logs structured as JSON messages. Filebeat processes the logs line by line, so the JSON decoding only works if there is one JSON object per message.

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


#### `container` [filebeat-input-filestream-parsers-container]

Use the `container` parser to extract information from  containers log files. It parses lines into common message lines, extracting timestamps too.

**`stream`**
:   Reads from the specified streams only: `all`, `stdout` or `stderr`. The default is `all`.

**`format`**
:   Use the given format when parsing logs: `auto`, `docker` or `cri`. The default is `auto`, it will automatically detect the format. To disable autodetection set any of the other options.

The following snippet configures Filebeat to read the `stdout` stream from all containers under the default Kubernetes logs path:

```yaml
  paths:
    - "/var/log/containers/*.log"
  parsers:
    - container:
        stream: stdout
```


#### `syslog` [_syslog]

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


#### `include_message` [_include_message]

Use the `include_message` parser to filter messages in the parsers pipeline. Messages that match the provided pattern are passed to the next parser, the others are dropped.

You should use `include_message` instead of `include_lines` if you would like to control when the filtering happens. `include_lines` runs after the parsers, `include_message` runs in the parsers pipeline.

**`patterns`**
:   List of regexp patterns to match.

This example shows you how to include messages that start with the string ERR or WARN:

```yaml
  paths:
    - "/var/log/containers/*.log"
  parsers:
    - include_message.patterns: ["^ERR", "^WARN"]
```


## Metrics [_metrics_8]

This input exposes metrics under the [HTTP monitoring endpoint](/reference/filebeat/http-endpoint.md). These metrics are exposed under the `/inputs` path. They can be used to observe the activity of the input. Note that metrics from processors are not included.

| Metric | Description |
| --- | --- |
| `files_opened_total` | Total number of files opened. |
| `files_closed_total` | Total number of files closed. |
| `files_active` | Number of files currently open (gauge). |
| `messages_read_total` | Total number of messages read. |
| `messages_truncated_total` | Total number of messages truncated. |
| `bytes_processed_total` | Total number of bytes processed. |
| `events_processed_total` | Total number of events processed. |
| `processing_errors_total` | Total number of processing errors. |
| `processing_time` | Histogram of the elapsed time to process messages (expressed in nanoseconds). |

Note:


## Common options [filebeat-input-filestream-common-options]

The following configuration options are supported by all inputs.


#### `enabled` [_enabled_9]

Use the `enabled` option to enable and disable inputs. By default, enabled is set to true.


#### `tags` [_tags_9]

A list of tags that Filebeat includes in the `tags` field of each published event. Tags make it easy to select specific events in Kibana or apply conditional filtering in Logstash. These tags will be appended to the list of tags specified in the general configuration.

Example:

```yaml
filebeat.inputs:
- type: filestream
  . . .
  tags: ["json"]
```


#### `fields` [filebeat-input-filestream-fields]

Optional fields that you can specify to add additional information to the output. For example, you might add fields that you can use for filtering log data. Fields can be scalar values, arrays, dictionaries, or any nested combination of these. By default, the fields that you specify here will be grouped under a `fields` sub-dictionary in the output document. To store the custom fields as top-level fields, set the `fields_under_root` option to true. If a duplicate field is declared in the general configuration, then its value will be overwritten by the value declared here.

```yaml
filebeat.inputs:
- type: filestream
  . . .
  fields:
    app_id: query_engine_12
```


#### `fields_under_root` [fields-under-root-filestream]

If this option is set to true, the custom [fields](#filebeat-input-filestream-fields) are stored as top-level fields in the output document instead of being grouped under a `fields` sub-dictionary. If the custom field names conflict with other field names added by Filebeat, then the custom fields overwrite the other fields.


#### `processors` [_processors_9]

A list of processors to apply to the input data.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.


#### `pipeline` [_pipeline_9]

The ingest pipeline ID to set for the events generated by this input.

::::{note}
The pipeline ID can also be configured in the Elasticsearch output, but this option usually results in simpler configuration files. If the pipeline is configured both in the input and output, the option from the input is used.
::::


::::{important}
The `pipeline` is always lowercased. If `pipeline: Foo-Bar`, then the pipeline name in {{es}} needs to be defined as `foo-bar`.
::::



#### `keep_null` [_keep_null_9]

If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.


#### `index` [_index_9]

If present, this formatted string overrides the index for events from this input (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"filebeat-myindex-2019.11.01"`.


#### `publisher_pipeline.disable_host` [_publisher_pipeline_disable_host_9]

By default, all events contain `host.name`. This option can be set to `true` to disable the addition of this field to all events. The default value is `false`.
