---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-module-fortinet.html
---

# Fortinet module [filebeat-module-fortinet]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/fortinet-intro.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This is a module for Fortinet logs sent in the syslog format. It supports the following devices:

* `firewall` fileset: Supports FortiOS Firewall logs.

To configure a remote syslog destination, please reference the [Fortigate/FortiOS Documentation](https://docs.fortinet.com/document/fortigate/6.0.0/cli-reference/260508/log-syslogd-syslogd2-syslogd3-syslogd4-setting).

The syslog format choosen should be `Default`.

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Compatibility [_compatibility_12]

This module has been tested against FortiOS version 6.0.x and 6.2.x. Versions above this are expected to work but have not been tested.


## Configure the module [configuring-fortinet-module]

You can further refine the behavior of the `fortinet` module by specifying [variable settings](#fortinet-settings) in the `modules.d/fortinet.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**


### Variable settings [fortinet-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `fortinet` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `fortinet.firewall.var.paths` instead of `firewall.var.paths`.
::::



### `firewall` fileset settings [_firewall_fileset_settings_2]

```yaml
- module: fortinet
  firewall:
    enabled: true
    var.input: udp
    var.syslog_host: 0.0.0.0
    var.syslog_port: 9004
```

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.


### Time zone support [_time_zone_support_5]

This module parses logs that don’t contain time zone information. For these logs, Filebeat reads the local time zone and uses it when parsing to convert the timestamp to UTC. The time zone to be used for parsing is included in the event in the `event.timezone` field.

To disable this conversion, the `event.timezone` field can be removed with the `drop_fields` processor.

If logs are originated from systems or applications with a different time zone to the local one, the `event.timezone` field can be overwritten with the original time zone using the `add_fields` processor.

See [Processors](/reference/filebeat/filtering-enhancing-data.md) for information about specifying processors in your config.

**`var.input`**
:   The input to use, can be either the value `tcp`, `udp` or `file`.

**`var.syslog_host`**
:   The interface to listen to all syslog traffic. Defaults to localhost. Set to 0.0.0.0 to bind to all available interfaces.

**`var.syslog_port`**
:   The port to listen for syslog traffic. Defaults to 9004.

**`var.tags`**
:   A list of tags to include in events. Including `forwarded` indicates that the events did not originate on this host and causes `host.name` to not be added to events. Defaults to `[fortinet-firewall, forwarded]`.


### Fortinet ECS fields [_fortinet_ecs_fields]

This is a list of FortiOS fields that are mapped to ECS.

| Fortinet Fields | ECS Fields |
| --- | --- |
| action | event.action |
| agent | user_agent.original |
| app | network.application |
| appcat | rule.category |
| applist | rule.ruleset |
| catdesc | rule.category |
| ccertissuer | tls.client_issuer |
| collectedemail | source.user.email |
| comment | rule.description |
| daddr | destination.address |
| devid | observer.serial_number |
| dir | network.direction |
| direction | network.direction |
| dst_host | destination.address |
| dstcollectedemail | destination.user.email |
| dst_int | observer.egress.interface.name |
| dstintf | observer.egress.interface.name |
| dstip | destination.ip |
| dstmac | destination.mac |
| dstname | destination.address |
| dst_port | destination.port |
| dstport | destination.port |
| dstunauthuser | destination.user.name |
| dtype | vulnerability.category |
| duration | event.duration |
| errorcode | error.code |
| event_id | event.id |
| eventid | event.id |
| eventtime | event.start |
| eventtype | event.action |
| file | file.name |
| filename | file.name |
| filesize | file.size |
| filetype | file.extension |
| filehash | file.hash.crc32 |
| from | source.user.email |
| group | source.user.group |
| hostname | url.domain |
| infectedfilename | file.name |
| infectedfilesize | file.size |
| infectedfiletype | file.extension |
| ipaddr | dns.resolved_ip |
| level | log.level |
| locip | source.ip |
| locport | source.port |
| logdesc | rule.description |
| logid | event.code |
| matchfilename | file.name |
| matchfiletype | file.extension |
| msg | message |
| error_num | error.code |
| policyid | rule.id |
| policy_id | rule.id |
| policyname | rule.name |
| policytype | rule.ruleset |
| poluuid | rule.uuid |
| profile | rule.ruleset |
| proto | network.iana_number |
| qclass | dns.question.class |
| qname | dns.question.name |
| qtype | dns.question.type |
| rcvdbyte | source.bytes |
| rcvdpkt | source.packets |
| recipient | destination.user.email |
| ref | event.reference |
| remip | destination.ip |
| remport | destination.port |
| saddr | source.address |
| scertcname | tls.client.server_name |
| scertissuer | tls.server.issuer |
| sender | source.user.email |
| sentbyte | source.bytes |
| sentpkt | source.packets |
| service | network.protocol |
| sess_duration | event.duration |
| srcdomain | source.domain |
| srcintf | observer.ingress.interface.name |
| srcip | source.ip |
| source_mac | source.mac |
| srcmac | source.mac |
| srcport | source.port |
| tranip | destination.nat.ip |
| tranport | destination.nat.port |
| transip | source.nat.ip |
| transport | source.nat.port |
| tz | event.timezone |
| unauthuser | source.user.name |
| url | url.path |
| user | source.user.name |
| xid | dns.id |


## Fields [_fields_17]

For a description of each field in the module, see the [exported fields](/reference/filebeat/exported-fields-fortinet.md) section.
