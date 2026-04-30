The `auditd` module receives audit events from the Linux Audit Framework that is a part of the Linux kernel.

This module is available only for Linux.


## How it works [_how_it_works]

This module establishes a subscription to the kernel to receive the events as they occur. So unlike most other modules, the `period` configuration option is unused because it is not implemented using polling.

The Linux Audit Framework can send multiple messages for a single auditable event. For example, a `rename` syscall causes the kernel to send eight separate messages. Each message describes a different aspect of the activity that is occurring (the syscall itself, file paths, current working directory, process title). This module will combine all of the data from each of the messages into a single event.

Messages for one event can be interleaved with messages from another event. This module will buffer the messages in order to combine related messages into a single event even if they arrive interleaved or out of order.


## Useful commands [_useful_commands]

When running Auditbeat with the `auditd` module enabled, you might find that other monitoring tools interfere with Auditbeat.

For example, you might encounter errors if another process, such as `auditd`, is registered to receive data from the Linux Audit Framework. You can use these commands to see if the `auditd` service is running and stop it:

* See if `auditd` is running:

    ```shell
    service auditd status
    ```

* Stop the `auditd` service:

    ```shell
    service auditd stop
    ```

* Disable `auditd` from starting on boot:

    ```shell
    chkconfig auditd off
    ```


To save CPU usage and disk space, you can use this command to stop `journald` from listening to audit messages:

```shell
systemctl mask systemd-journald-audit.socket
```


## Inspect the kernel audit system status [_inspect_the_kernel_audit_system_status]

Auditbeat provides useful commands to query the state of the audit system in the Linux kernel.

* See the list of installed audit rules:

    ```shell
    auditbeat show auditd-rules
    ```

    Prints the list of loaded rules, similar to `auditctl -l`:

    ```shell
    -a never,exit -S all -F pid=26253
    -a always,exit -F arch=b32 -S all -F key=32bit-abi
    -a always,exit -F arch=b64 -S execve,execveat -F key=exec
    -a always,exit -F arch=b64 -S connect,accept,bind -F key=external-access
    -w /etc/group -p wa -k identity
    -w /etc/passwd -p wa -k identity
    -w /etc/gshadow -p wa -k identity
    -a always,exit -F arch=b64 -S open,truncate,ftruncate,creat,openat,open_by_handle_at -F exit=-EACCES -F key=access
    -a always,exit -F arch=b64 -S open,truncate,ftruncate,creat,openat,open_by_handle_at -F exit=-EPERM -F key=access
    ```

* See the status of the audit system:

    ```shell
    auditbeat show auditd-status
    ```

    Prints the status of the kernel audit system, similar to `auditctl -s`:

    ```shell
    enabled 1
    failure 0
    pid 0
    rate_limit 0
    backlog_limit 8192
    lost 14407
    backlog 0
    backlog_wait_time 0
    features 0xf
    ```



## Configuration options [_configuration_options_17]

This module has some configuration options for tuning its behavior. The following example shows all configuration options with their default values.

```yaml
- module: auditd
  resolve_ids: true
  failure_mode: silent
  backlog_limit: 8192
  rate_limit: 0
  include_raw_message: false
  include_warnings: false
  backpressure_strategy: auto
  immutable: false
```

This module also supports the [standard configuration options](#module-standard-options-auditd) described later.

**`socket_type`**
:   This optional setting controls the type of socket that Auditbeat uses to receive events from the kernel. The two options are `unicast` and `multicast`.

    `unicast` should be used when Auditbeat is the primary userspace daemon for receiving audit events and managing the rules. Only a single process can receive audit events through the "unicast" connection so any other daemons should be stopped (e.g. stop `auditd`).

    `multicast` can be used in kernel versions 3.16 and newer. By using `multicast` Auditbeat will receive an audit event broadcast that is not exclusive to a a single process. This is ideal for situations where `auditd` is running and managing the rules.

    By default Auditbeat will use `multicast` if the kernel version is 3.16 or newer and no rules have been defined. Otherwise `unicast` will be used.


**`immutable`**
:   This boolean setting sets the audit config as immutable (`-e 2`). This option can only be used with the `socket_type: unicast` since Auditbeat needs to manage the rules to be able to set it.

    It is important to note that with this setting enabled, if Auditbeat is stopped and resumed events will continue to be processed but the configuration won’t be updated until the system is restarted entirely.


**`resolve_ids`**
:   This boolean setting enables the resolution of UIDs and GIDs to their associated names. The default value is true.

**`failure_mode`**
:   This determines the kernel’s behavior on critical failures such as errors sending events to Auditbeat, the backlog limit was exceeded, the kernel ran out of memory, or the rate limit was exceeded. The options are `silent`, `log`, or `panic`. `silent` basically makes the kernel ignore the errors, `log` makes the kernel write the audit messages using `printk` so they show up in system’s syslog, and `panic` causes the kernel to panic to prevent use of the machine. Auditbeat’s default is `silent`.

**`backlog_limit`**
:   This controls the maximum number of audit messages that will be buffered by the kernel.

**`rate_limit`**
:   This sets a rate limit on the number of messages/sec delivered by the kernel. The default is 0, which disables rate limiting. Changing this value to anything other than zero can cause messages to be lost. The preferred approach to reduce the messaging rate is be more selective in the audit ruleset.

**`include_raw_message`**
:   This boolean setting causes Auditbeat to include each of the raw messages that contributed to the event in the document as a field called `event.original`. The default value is false. This setting is primarily used for development and debugging purposes.

**`include_warnings`**
:   This boolean setting causes Auditbeat to include as warnings any issues that were encountered while parsing the raw messages. The messages are written to the `error.message` field. The default value is false. When this setting is enabled the raw messages will be included in the event regardless of the `include_raw_message` config setting. This setting is primarily used for development and debugging purposes.

**`audit_rules`**
:   A string containing the audit rules that should be installed to the kernel. There should be one rule per line. Comments can be embedded in the string using `#` as a prefix. The format for rules is the same used by the Linux `auditctl` utility. Auditbeat supports adding file watches (`-w`) and syscall rules (`-a` or `-A`). For more information, see [Audit rules](#audit-rules).

**`audit_rule_files`**
:   A list of files to load audit rules from. This files are loaded after the rules declared in `audit_rules` are loaded. Wildcards are supported and will expand in lexicographical order. The format is the same as that of the `audit_rules` field.

**`ignore_errors`**
:   This setting allows errors during rule loading and parsing to be ignored, but logged as warnings.

**`backpressure_strategy`**
:   Specifies the strategy that Auditbeat uses to prevent backpressure from propagating to the kernel and impacting audited processes.

    The possible values are:

    * `auto` (default): Auditbeat uses the `kernel` strategy, if supported, or falls back to the `userspace` strategy.
    * `kernel`: Auditbeat sets the `backlog_wait_time` in the kernel’s audit framework to 0. This causes events to be discarded in the kernel if the audit backlog queue fills to capacity. Requires a 3.14 kernel or newer.
    * `userspace`: Auditbeat drops events when there is backpressure from the publishing pipeline. If no `rate_limit` is set, Auditbeat sets a rate limit of 5000. Users should test their setup and adjust the `rate_limit` option accordingly.
    * `both`: Auditbeat uses the `kernel` and `userspace` strategies at the same time.
    * `none`: No backpressure mitigation measures are enabled.



### Standard configuration options [module-standard-options-auditd]

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


## Audit rules [audit-rules]

The audit rules are where you configure the activities that are audited. These rules are configured as either syscalls or files that should be monitored. For example you can track all `connect` syscalls or file system writes to `/etc/passwd`.

Auditing a large number of syscalls can place a heavy load on the system so consider carefully the rules you define and try to apply filters in the rules themselves to be as selective as possible.

The kernel evaluates the rules in the order in which they were defined so place the most active rules first in order to speed up evaluation.

You can assign keys to each rule for better identification of the rule that triggered an event and easier filtering later in Elasticsearch.

Defining any audit rules in the config causes Auditbeat to purge all existing audit rules prior to adding the rules specified in the config. Therefore it is unnecessary and unsupported to include a `-D` (delete all) rule.

```sh
auditbeat.modules:
- module: auditd
  audit_rules: |
    # Things that affect identity.
    -w /etc/group -p wa -k identity
    -w /etc/passwd -p wa -k identity
    -w /etc/gshadow -p wa -k identity
    -w /etc/shadow -p wa -k identity

    # Unauthorized access attempts to files (unsuccessful).
    -a always,exit -F arch=b32 -S open,creat,truncate,ftruncate,openat,open_by_handle_at -F exit=-EACCES -F auid>=1000 -F auid!=4294967295 -F key=access
    -a always,exit -F arch=b32 -S open,creat,truncate,ftruncate,openat,open_by_handle_at -F exit=-EPERM -F auid>=1000 -F auid!=4294967295 -F key=access
    -a always,exit -F arch=b64 -S open,truncate,ftruncate,creat,openat,open_by_handle_at -F exit=-EACCES -F auid>=1000 -F auid!=4294967295 -F key=access
    -a always,exit -F arch=b64 -S open,truncate,ftruncate,creat,openat,open_by_handle_at -F exit=-EPERM -F auid>=1000 -F auid!=4294967295 -F key=access
```
