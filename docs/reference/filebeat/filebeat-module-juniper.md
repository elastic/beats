---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-module-juniper.html
---

# Juniper module [filebeat-module-juniper]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This is a module for ingesting data from the different Juniper Products. Currently supports these filesets:

* `srx` fileset: Supports Juniper SRX logs

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Configure the module [configuring-juniper-module]

You can further refine the behavior of the `juniper` module by specifying [variable settings](#juniper-settings) in the `modules.d/juniper.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**


### Variable settings [juniper-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `juniper` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `juniper.{{fileset_ex}}.var.paths` instead of `{{fileset_ex}}.var.paths`.
::::


::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::



### `srx` fileset settings [_srx_fileset_settings]

The Juniper-SRX module only supports syslog messages in the format "structured-data + brief" [JunOS Documentation structured-data](https://www.juniper.net/documentation/en_US/junos/topics/reference/configuration-statement/structured-data-edit-system.html)

To configure a remote syslog destination, please reference the [SRX Getting Started - Configure System Logging](https://kb.juniper.net/InfoCenter/index?page=content&id=kb16502).

The following processes and tags are supported:

| JunOS processes | JunOS tags |  |
| --- | --- | --- |
| RT_FLOW | RT_FLOW_SESSION_CREATE |  |
|  | RT_FLOW_SESSION_CLOSE |  |
|  | RT_FLOW_SESSION_DENY |  |
|  | APPTRACK_SESSION_CREATE |  |
|  | APPTRACK_SESSION_CLOSE |  |
|  | APPTRACK_SESSION_VOL_UPDATE |  |
| RT_IDS | RT_SCREEN_TCP |  |
|  | RT_SCREEN_UDP |  |
|  | RT_SCREEN_ICMP |  |
|  | RT_SCREEN_IP |  |
|  | RT_SCREEN_TCP_DST_IP |  |
|  | RT_SCREEN_TCP_SRC_IP |  |
| RT_UTM | WEBFILTER_URL_PERMITTED |  |
|  | WEBFILTER_URL_BLOCKED |  |
|  | AV_VIRUS_DETECTED_MT |  |
|  | CONTENT_FILTERING_BLOCKED_MT |  |
|  | ANTISPAM_SPAM_DETECTED_MT |  |
| RT_IDP | IDP_ATTACK_LOG_EVENT |  |
|  | IDP_APPDDOS_APP_STATE_EVENT |  |
| RT_AAMW | SRX_AAMW_ACTION_LOG |  |
|  | AAMW_MALWARE_EVENT_LOG |  |
|  | AAMW_HOST_INFECTED_EVENT_LOG |  |
|  | AAMW_ACTION_LOG |  |
| RT_SECINTEL | SECINTEL_ACTION_LOG |  |

The syslog format choosen should be `Default`.


## Compatibility [_compatibility_18]

This module has been tested against JunOS version 19.x and 20.x. Versions above this are expected to work but have not been tested.

```yaml
- module: juniper
  junos:
    enabled: true
    var.input: udp
    var.syslog_host: 0.0.0.0
    var.syslog_port: 9006
```

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.

**`var.input`**
:   The input to use, can be either the value `tcp`, `udp` or `file`.

**`var.syslog_host`**
:   The interface to listen to all syslog traffic. Defaults to localhost. Set to 0.0.0.0 to bind to all available interfaces.

**`var.syslog_port`**
:   The port to listen for syslog traffic. Defaults to 9006.


### Juniper SRX ECS fields [_juniper_srx_ecs_fields]

This is a list of JunOS fields that are mapped to ECS.

| Juniper SRX Fields | ECS Fields |  |
| --- | --- | --- |
| application-risk | event.risk_score |  |
| bytes-from-client | source.bytes |  |
| bytes-from-server | destination.bytes |  |
| destination-interface-name | observer.egress.interface.name |  |
| destination-zone-name | observer.egress.zone |  |
| destination-address | destination.ip |  |
| destination-port | destination.port |  |
| dst_domainname | url.domain |  |
| elapsed-time | event.duration |  |
| filename | file.name |  |
| nat-destination-address | destination.nat.ip |  |
| nat-destination-port | destination.nat.port |  |
| nat-source-address | source.nat.ip |  |
| nat-source-port | source.nat.port |  |
| message | message |  |
| obj | url.path |  |
| packets-from-client | source.packets |  |
| packets-from-server | destination.packets |  |
| policy-name | rule.name |  |
| protocol | network.transport |  |
| source-address | source.ip |  |
| source-interface-name | observer.ingress.interface.name |  |
| source-port | source.port |  |
| source-zone-name | observer.ingress.zone |  |
| url | url.domain |  |


## Fields [_fields_25]

For a description of each field in the module, see the [exported fields](/reference/filebeat/exported-fields-juniper.md) section.
