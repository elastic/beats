---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-module-sophos.html
---

# Sophos module [filebeat-module-sophos]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/sophos/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This is a module for Sophos Products, currently it accepts logs in syslog format or from a file for the following devices:

* `xg` fileset: supports Sophos XG SFOS logs.

To configure a remote syslog destination, please reference the [SophosXG/SFOS Documentation](https://docs.sophos.com/nsg/sophos-firewall/18.5/Help/en-us/webhelp/onlinehelp/nsg/tasks/SyslogServerAdd.md).

The syslog format choosen in Sophos configuration should be `Central Reporting Format`.

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Compatibility [_compatibility_35]

This module has been tested against SFOS version 17.5.x, 18.0.x, and 18.5.x. Versions above this and between 18.0 - 18.5 are expected to work but have not been tested.


## Configure the module [configuring-sophos-module]

You can further refine the behavior of the `sophos` module by specifying [variable settings](#sophos-settings) in the `modules.d/sophos.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**


### Variable settings [sophos-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `sophos` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `sophos.xg.var.paths` instead of `xg.var.paths`.
::::



### `xg` fileset settings [_xg_fileset_settings]

The Sophos XG firewalls do not include hostname in either the syslog header or body, and the only unique identifier for each firewall is the related serial number.

Below you will see an example configuration file, that sets the default hostname (if no serial number is included in the config file), and example on how to map serial numbers to a hostname

```yaml
- module: sophos
  xg:
    enabled: true
    var.input: udp
    var.syslog_host: 0.0.0.0
    var.syslog_port: 9005
    var.default_host_name: firewall.localgroup.local
    var.known_devices:
      - serial_number: "1234567890123457"
        hostname: "a.host.local"
      - serial_number: "1234234590678557"
        hostname: "b.host.local"
```

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.

**`var.input`**
:   The input to use, can be either the value `tcp`, `udp` or `file`.

**`var.syslog_host`**
:   The interface to listen to all syslog traffic. Defaults to localhost. Set to 0.0.0.0 to bind to all available interfaces.

**`var.syslog_port`**
:   The port to listen for syslog traffic. Defaults to 9005.

**`var.host_name`**
:   Host name / Observer name, since SophosXG does not provide this in the syslog file. Default to `firewall.localgroup.local`


### SophosXG ECS fields [_sophosxg_ecs_fields]

This is a list of SophosXG fields that are mapped to ECS.

| SophosXG Fields | ECS Fields |  |
| --- | --- | --- |
| application | network.protocol |  |
| classification | rule.category |  |
| device_id | observer.serial_number |  |
| domainname | url.domain |  |
| dst_host | destination.address |  |
| dst_int | observer.egress.interface.name |  |
| dstzonetype | observer.egress.zone |  |
| dst_ip | destination.ip |  |
| destinationip | destination.ip |  |
| dst_mac | destination.mac |  |
| dstname | destination.address |  |
| dst_port | destination.port |  |
| dst_domainname | url.domain |  |
| duration | event.duration |  |
| filename | file.name |  |
| filetype | file.extension |  |
| file_size | file.size |  |
| file_path | file.directory |  |
| fw_rule_id | rule.id |  |
| from_email_address | source.user.email |  |
| httpstatus | http.response.status_code |  |
| in_interface | observer.ingress.interface.name |  |
| log_id | event.code |  |
| log_subtype | event.action |  |
| message | message |  |
| method | http.request.method |  |
| policy_type | rule.ruleset |  |
| protocol | network.transport |  |
| recv_bytes | destination.bytes |  |
| recv_pkts | destination.packets |  |
| referer | http.request.referrer |  |
| sent_bytes | source.bytes |  |
| sent_pkts | source.packets |  |
| sha1sum | file.hash.sha1 |  |
| srczonetype | observer.ingress.zone |  |
| src_ip | source.ip |  |
| src_domainname | url.domain |  |
| sourceip | source.ip |  |
| src_mac | source.mac |  |
| src_port | source.port |  |
| status_code | http.response.status_code |  |
| time_zone | event.timezone |  |
| to_email_address | destination.user.email |  |
| tran_dst_ip | destination.nat.ip |  |
| tran_dst_port | destination.nat.port |  |
| tran_src_ip | source.nat.ip |  |
| tran_src_port | source.nat.port |  |
| url | url.original |  |
| user_agent | user_agent.original |  |
| useragent | user_agent.original |  |
| user_gp | source.user.group |  |
| user_name | source.user.name |  |
| ws_protocol | http.version |  |


## Fields [_fields_50]

For a description of each field in the module, see the [exported fields](/reference/filebeat/exported-fields-sophos.md) section.
