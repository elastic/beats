---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-module-checkpoint.html
---

# Check Point module [filebeat-module-checkpoint]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/checkpoint/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This is a module for Check Point firewall logs. It supports logs from the Log Exporter in the Syslog RFC 5424 format. If you need to ingest Check Point logs in CEF format then please use the [`CEF module`](/reference/filebeat/filebeat-module-cef.md) (more fields are provided in the syslog output).

To configure a Log Exporter, please refer to the documentation by [Check Point](https://supportcenter.checkpoint.com/supportcenter/portal?eventSubmit_doGoviewsolutiondetails=&solutionid=sk122323).

Example Log Exporter config:

`cp_log_export add name testdestination target-server 192.168.1.1 target-port 9001 protocol udp format syslog`

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Compatibility [_compatibility_7]

This module has been tested against Check Point Log Exporter on R80.X but should also work with R77.30.


## Configure the module [configuring-checkpoint-module]

You can further refine the behavior of the `checkpoint` module by specifying [variable settings](#checkpoint-settings) in the `modules.d/checkpoint.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**


### Variable settings [checkpoint-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `checkpoint` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `checkpoint.firewall.var.paths` instead of `firewall.var.paths`.
::::



### `firewall` fileset settings [_firewall_fileset_settings]

Example config:

```yaml
- module: checkpoint
  firewall:
    var.syslog_host: 0.0.0.0
    var.syslog_port: 9001
```

**`var.paths`**
:   An array of glob-based paths that specify where to look for the log files. All patterns supported by [Go Glob](https://golang.org/pkg/path/filepath/#Glob) are also supported here. For example, you can use wildcards to fetch all files from a predefined level of subdirectories: `/path/to/log/*/*.log`. This fetches all `.log` files from the subfolders of `/path/to/log`. It does not fetch log files from the `/path/to/log` folder itself. If this setting is left empty, Filebeat will choose log paths based on your operating system.

**`var.syslog_host`**
:   The interface to listen to UDP based syslog traffic. Defaults to localhost. Set to 0.0.0.0 to bind to all available interfaces.

**`var.syslog_port`**
:   The UDP port to listen for syslog traffic. Defaults to 9001.

**`var.timezone_offset`**
:   IANA time zone or time offset (e.g. `+0200`) to use when interpreting syslog timestamps without a time zone. Defaults to UTC.

**`var.tags`**
:   A list of tags to include in events. Including `forwarded` indicates that the events did not originate on this host and causes `host.name` to not be added to events. Defaults to `[checkpoint-firewall, forwarded]`.

**`var.ssl`**
:   The SSL/TLS configuration for the filebeat instance. This can be used to enforce mutual TLS.

```yaml
ssl:
  enabled: true
  certificate_authorities: ["my-ca.pem"]
  certificate: "filebeat-cert.pem"
  key: "filebeat-key.pem"
  client_authentication: "required"
```


### Check Point devices [_check_point_devices_2]

This module will parse Check Point Syslog data as documented in: [Checkpoint Log Fields Description.](https://supportcenter.checkpoint.com/supportcenter/portal?eventSubmit_doGoviewsolutiondetails=&solutionid=sk144192)

Check Point Syslog extensions are mapped as follows to ECS:

| Check Point Fields | ECS Fields |  |
| --- | --- | --- |
| action | event.action |  |
| appi_name | network.application |  |
| app_risk | event.risk_score |  |
| app_rule_id | rule.id |  |
| app_rule_name | rule.name |  |
| bytes | network.bytes |  |
| categories | rule.category |  |
| client_inbound_interface | observer.ingress.interface.name |  |
| client_outbound_bytes | source.bytes |  |
| client_outbound_interface | observer.egress.interface.name |  |
| client_outbound_packets | source.packets |  |
| destination_dns_hostname | destination.domain |  |
| dlp_file_name | file.name |  |
| dns_message_type | dns.type |  |
| dns_type | dns.question.type |  |
| domain_name | dns.question.name |  |
| dst | destination.ip |  |
| dst_machine_name | destination.domain |  |
| dlp_rule_name | rule.name |  |
| dlp_rule_uid | rule.uuid |  |
| endpoint_ip | observer.ip |  |
| file_id | file.inode |  |
| file_type | file.type |  |
| file_name | file.name |  |
| file_size | file.size |  |
| file_md5 | file.hash.md5 |  |
| file_sha1 | file.hash.sha1 |  |
| file_sha256 | file.hash.sha256 |  |
| first_detection | event.start |  |
| from | source.user.email |  |
| ifdir | network.direction |  |
| industry_reference | vulnerability.id |  |
| inzone | observer.ingress.zone |  |
| last_detection | event.end |  |
| loguid | event.id |  |
| mac_destination_address | destination.mac |  |
| mac_source_address | source.mac |  |
| malware_action | rule.description |  |
| matched_category | rule.category |  |
| malware_rule_id | rule.rule.id |  |
| message | message |  |
| method | http.request.method |  |
| origin | observer.name |  |
| origin_ip | observer.ip |  |
| os_name | host.os.name |  |
| os_version | host.os.version |  |
| outzone | observer.egress.zone |  |
| packet_capture | event.url |  |
| packets | network.packets |  |
| parent_process_md5 | process.parent.hash.md5 |  |
| parent_process_name | process.parent.name |  |
| process_md5 | process.hash.md5 |  |
| process_name | process.name |  |
| product | observer.product |  |
| proto | network.iana_number |  |
| reason | message |  |
| received_bytes | destination.bytes |  |
| referrer | http.request.referrer |  |
| rule_name | rule.name |  |
| resource | url.original |  |
| s_port | source.port |  |
| security_inzone | observer.ingress.zone |  |
| security_outzone | observer.egress.zone |  |
| sent_bytes | source.bytes |  |
| sequencenum | event.sequence |  |
| service | destination.port |  |
| service_id | network.application |  |
| service_name | destination.service.name |  |
| server_outbound_packets | destination.packets |  |
| server_outbound_bytes | destination.bytes |  |
| severity | event.severity |  |
| smartdefense_profile | rule.ruleset |  |
| src | source.ip |  |
| src_machine_name | source.domain |  |
| src_user_group | source.user.group.name |  |
| start_time | event.start |  |
| status | http.response.status_code |  |
| tid | dns.id |  |
| time | @timestamp |  |
| to | destination.user.email |  |
| type | observer.type |  |
| update_version | observer.version |  |
| url | url.original |  |
| user_group | group.name |  |
| usercheck_incident_uid | destination.user.id |  |
| web_client_type | user_agent.name |  |
| xlatesrc | source.nat.ip |  |
| xlatedst | destination.nat.ip |  |
| xlatesport | source.nat.port |  |
| xlatedport | destination.nat.port |  |


## Fields [_fields_10]

For a description of each field in the module, see the [exported fields](/reference/filebeat/exported-fields-checkpoint.md) section.
