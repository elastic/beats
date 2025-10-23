The `file_integrity` module sends events when a file is changed (created, updated, or deleted) on disk. The events contain file metadata and hashes.

The module is implemented for Linux, macOS (Darwin), and Windows.


## How it works [_how_it_works_2]

This module uses features of the operating system to monitor file changes in realtime. When the module starts it creates a subscription with the OS to receive notifications of changes to the specified files or directories. Upon receiving notification of a change the module will read the file’s metadata and then compute a hash of the file’s contents.

At startup this module will perform an initial scan of the configured files and directories to generate baseline data for the monitored paths and detect changes since the last time it was run. It uses locally persisted data in order to only send events for new or modified files.

The operating system features that power this feature are as follows.

* Linux - Multiple backends are supported: `auto`, `fsnotify`, `kprobes`, `ebpf`. By default, `fsnotify` is used, and therefore the kernel must have inotify support. Inotify was initially merged into the 2.6.13 Linux kernel. The eBPF backend uses modern eBPF features and supports 5.10.16+ kernels. The `Kprobes` backend uses tracefs and supports 3.10+ kernels. FSNotify doesn’t have the ability to associate user data to file events. The preferred backend can be selected by specifying the `backend` config option. Since eBPF and Kprobes are in technical preview, `auto` will default to `fsnotify`.
* macOS (Darwin) - Uses the `FSEvents` API, present since macOS 10.5. This API coalesces multiple changes to a file into a single event. Auditbeat translates this coalesced changes into a meaningful sequence of actions. However, in rare situations the reported events may have a different ordering than what actually happened.
* Windows:
  * `ReadDirectoryChangesW` is used.
  * {applies_to}`stack: preview 9.2.0` Multiple backends are supported: `auto`, `fsnotify`, `etw`. By default, `fsnotify` is used, which utilizes the `ReadDirectoryChangesW` Windows API. The `etw` backend uses Event Tracing for Windows (ETW) to monitor file system activities at the kernel level, supporting enhanced process context information. It requires Administrator privileges.

The file integrity module should not be used to monitor paths on network file systems.


## Configuration options [_configuration_options_18]

This module has some configuration options for tuning its behavior. The following example shows all configuration options with their default values for Linux.

```yaml
- module: file_integrity
  paths:
  - /bin
  - /usr/bin
  - /sbin
  - /usr/sbin
  - /etc
  recursive: false
  exclude_files:
  - '(?i)\.sw[nop]$'
  - '~$'
  - '/\.git($|/)'
  include_files: []
  scan_at_start: true
  scan_rate_per_sec: 50 MiB
  max_file_size: 100 MiB
  hash_types: [sha1]
```

{applies_to}`stack: preview 9.2.0` For Windows with the ETW backend, additional configuration options are available:

```yaml
- module: file_integrity
  paths:
  - C:\Windows\System32
  - C:\Program Files
  backend: etw
  recursive: true
  flush_interval: 1m
```

This module also supports the [standard configuration options](#module-standard-options-file_integrity) described later.

**`paths`**
:   A list of paths (directories or files) to watch. Globs are not supported. The specified paths should exist when the metricset is started. Paths should be absolute, although the file integrity module will attempt to resolve relative path events to their absolute file path. Symbolic links will be resolved on module start and the link target will be watched if link resolution is successful. Changes to the symbolic link after module start will not change the watch target. If the link does not resolve to a valid target, the symbolic link itself will be watched; if the symlink target becomes valid after module start up this will not be picked up by the file system watches.

**`recursive`**
:   By default, the watches set to the paths specified in `paths` are not recursive. This means that only changes to the contents of this directories are watched. If `recursive` is set to `true`, the `file_integrity` module will watch for changes on this directory and all its subdirectories.

**`exclude_files`**
:   A list of regular expressions used to filter out events for unwanted files. The expressions are matched against the full path of every file and directory. When used in conjunction with `include_files`, file paths need to match both `include_files` and not match `exclude_files` to be selected. By default, no files are excluded. See [*Regular expression support*](/reference/auditbeat/regexp-support.md) for a list of supported regexp patterns. It is recommended to wrap regular expressions in single quotation marks to avoid issues with YAML escaping rules. If `recursive` is set to true, subdirectories can also be excluded here by specifying them.

**`include_files`**
:   A list of regular expressions used to specify which files to select. When configured, only files matching the pattern will be monitored. The expressions are matched against the full path of every file and directory. When used in conjunction with `exclude_files`, file paths need to match both `include_files` and not match `exclude_files` to be selected. By default, all files are selected. See [*Regular expression support*](/reference/auditbeat/regexp-support.md) for a list of supported regexp patterns. It is recommended to wrap regular expressions in single quotation marks to avoid issues with YAML escaping rules.

**`scan_at_start`**
:   A boolean value that controls if Auditbeat scans over the configured file paths at startup and send events for the files that have been modified since the last time Auditbeat was running. The default value is true.

    This feature depends on data stored locally in `path.data` in order to determine if a file has changed. The first time Auditbeat runs it will send an event for each file it encounters.


**`scan_rate_per_sec`**
:   When `scan_at_start` is enabled this sets an average read rate defined in bytes per second for the initial scan. This throttles the amount of CPU and I/O that Auditbeat consumes at startup. The default value is "50 MiB". Setting the value to "0" disables throttling. For convenience units can be specified as a suffix to the value. The supported units are `b` (default), `kib`, `kb`, `mib`, `mb`, `gib`, `gb`, `tib`, `tb`, `pib`, `pb`, `eib`, and `eb`.

**`max_file_size`**
:   The maximum size of a file in bytes for which Auditbeat will compute hashes and run file parsers. Files larger than this size will not be hashed or analysed by configured file parsers. The default value is 100 MiB. For convenience, units can be specified as a suffix to the value. The supported units are `b` (default), `kib`, `kb`, `mib`, `mb`, `gib`, `gb`, `tib`, `tb`, `pib`, `pb`, `eib`, and `eb`.

**`hash_types`**
:   A list of hash types to compute when the file changes. The supported hash types are `blake2b_256`, `blake2b_384`, `blake2b_512`, `md5`, `sha1`, `sha224`, `sha256`, `sha384`, `sha512`, `sha512_224`, `sha512_256`, `sha3_224`, `sha3_256`, `sha3_384`, `sha3_512`, and `xxh64`. The default value is `sha1`.

**`file_parsers`**
:   A list of `file_integrity` fields under `file` that will be populated by file format parsers. The available fields that can be analysed are listed in the auditbeat.reference.yml file. File parsers are run on all files within the `max_file_size` limit in the configured paths during a scan or when a file event involves the file. Files that are not targets of the specific file parser are only sniffed to examine whether analysis should proceed. This will usually only involve reading a small number of bytes.

**`backend`**
:   Select the backend that will be used to source events. The available backends vary by operating system:
    
    * **Linux:** `auto`, `fsnotify`, `kprobes`, `ebpf`. Default: `fsnotify`.
    * {applies_to}`stack: ga 9.2.0` **Windows:** `auto`, `fsnotify`, `etw`. Default: `fsnotify`.
    * {applies_to}`stack: ga 9.2.0` **macOS:** Only `auto` and `fsnotify` are supported. Default: `fsnotify`

**`flush_interval`** {applies_to}`stack: ga 9.2.0`
:   (**ETW backend only**) Controls how often the ETW backend flushes event correlation groups. The ETW backend groups related file operations (like create, write, close) to provide meaningful events. This setting determines how long to wait for related events before considering an operation complete and sending the event. Setting a shorter interval will send events more quickly but may break up related operations. Setting a longer interval will provide better event correlation but may delay event delivery and impact memory footprint. This option is ignored when using other backends. Default: `1m`.


### Standard configuration options [module-standard-options-file_integrity]

You can specify the following options for any Auditbeat module.

**`module`**
:   The name of the module to run.

**`enabled`**
:   A Boolean value that specifies whether the module is enabled.

**`fields`**
:   A dictionary of fields that will be sent with the dataset event. This setting is optional.

**`tags`**
:   A list of tags that will be sent with the dataset event. This setting is optional.

**`processors`**
:   A list of processors to apply to the data generated by the dataset.

    See [Processors](/reference/auditbeat/filtering-enhancing-data.md) for information about specifying processors in your config.


**`index`**
:   If present, this formatted string overrides the index for events from this module (for elasticsearch outputs), or sets the `raw_index` field of the event’s metadata (for other outputs). This string can only refer to the agent name and version and the event timestamp; for access to dynamic fields, use `output.elasticsearch.index` or a processor.

    Example value: `"%{[agent.name]}-myindex-%{+yyyy.MM.dd}"` might expand to `"auditbeat-myindex-2019.12.13"`.


**`keep_null`**
:   If this option is set to true, fields with `null` values will be published in the output document. By default, `keep_null` is set to `false`.

**`service.name`**
:   A name given by the user to the service the data is collected from. It can be used for example to identify information collected from nodes of different clusters with the same `service.type`.
