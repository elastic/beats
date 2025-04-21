---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-module-cef.html
---

# CEF module [filebeat-module-cef]

:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/cef/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This is a module for receiving Common Event Format (CEF) data over Syslog. When messages are received over the syslog protocol the syslog input will parse the header and set the timestamp value. Then the [`decode_cef`](/reference/filebeat/processor-decode-cef.md) processor is applied to parse the CEF encoded data. The decoded data is written into a `cef` object field. Lastly any Elastic Common Schema (ECS) fields that can be populated with the CEF data are populated.

::::{tip}
Read the [quick start](/reference/filebeat/filebeat-installation-configuration.md) to learn how to configure and run modules.
::::



## Configure the module [configuring-cef-module]

You can further refine the behavior of the `cef` module by specifying [variable settings](#cef-settings) in the `modules.d/cef.yml` file, or overriding settings at the command line.

You must enable at least one fileset in the module. **Filesets are disabled by default.**


### Variable settings [cef-settings]

Each fileset has separate variable settings for configuring the behavior of the module. If you don’t specify variable settings, the `cef` module uses the defaults.

For advanced use cases, you can also override input settings. See [Override input settings](/reference/filebeat/advanced-settings.md).

::::{tip}
When you specify a setting at the command line, remember to prefix the setting with the module name, for example, `cef.log.var.paths` instead of `log.var.paths`.
::::



### `log` fileset settings [_log_fileset_settings_2]

**`var.syslog_host`**
:   The interface to listen to UDP based syslog traffic. Defaults to `localhost`. Set to `0.0.0.0` to bind to all available interfaces.

**`var.syslog_port`**
:   The UDP port to listen for syslog traffic. Defaults to `9003`

::::{note}
Ports below 1024 require Filebeat to run as root.
::::


**`var.tags`**
:   A list of tags to include in events. Including `forwarded` indicates that the events did not originate on this host and causes `host.name` to not be added to events. Defaults to `[cef, forwarded]`.

**`var.timezone`**
:   IANA time zone name (e.g. `America/New_York`) or fixed time offset (e.g. `+0200`) to use when parsing times from the CEF message that do not contain a time zone. `Local` may be specified to use the machine’s local time zone. Defaults to `UTC`.


### Forcepoint NGFW Security Management Center [_forcepoint_ngfw_security_management_center]

This module will process CEF data from Forcepoint NGFW Security Management Center (SMC).  In the SMC configure the logs to be forwarded to the address set in `var.syslog_host` in format CEF and service UDP on `var.syslog_port`.  Instructions can be found in [KB 15002](https://support.forcepoint.com/KBArticle?id=000015002) for configuring the SMC.  Testing was done with CEF logs from SMC version 6.6.1 and custom string mappings were taken from *CEF Connector Configuration Guide* dated December 5, 2011.


### Check Point devices [_check_point_devices]

This module will parse CEF data form Check Point devices as documented in [Log Exporter CEF Field Mappings.](https://community.checkpoint.com/t5/Logging-and-Reporting/Log-Exporter-CEF-Field-Mappings/td-p/41060)

Check Point CEF extensions are mapped as follows:

| CEF Extension | CEF Label value | ECS Fields | Non-ECS Field |  |
| --- | --- | --- | --- | --- |
| cp_app_risk | - | event.risk_score | checkpoint.app_risk |  |
| cp_severity | - | event.severity | checkpoint.severity |  |
| baseEventCount | - | - | checkpoint.event_count |  |
| deviceExternalId | - | observer.type | - |  |
| deviceFacility | - | observer.type | - |  |
| deviceInboundInterface | - | observer.ingress.interface.name | - |  |
| deviceOutboundInterface | - | observer.egress.interface.name | - |  |
| externalId | - | - | checkpoint.uuid |  |
| fileHash | - | file.hash.{md5,sha1} | - |  |
| reason | - | - | checkpoint.termination_reason |  |
| requestCookies | - | - | checkpoint.cookie |  |
| sourceNtDomain | - | dns.question.name | - |  |
| Signature | - | vulnerability.id | - |  |
| Recipient | - | destination.user.email | - |  |
| Sender | - | source.user.email | - |  |
| deviceCustomFloatingPoint1 | update version | observer.version | - |  |
| deviceCustomIPv6Address2 | source ipv6 address | source.ip | - |  |
| deviceCustomIPv6Address3 | destination ipv6 address | destination.ip | - |  |
| deviceCustomNumber1 | elapsed time in seconds | event.duration | - |  |
| email recipients number | - | checkpoint.email_recipients_num |  |
| payload | network.bytes | - |  |
| deviceCustomNumber2 | icmp type | - | checkpoint.icmp_type |  |
| duration in seconds | event.duration | - |  |
| deviceCustomNumber3 | icmp code | - | checkpoint.icmp_code |  |
| deviceCustomString1 | connectivity state | - | checkpoint.connectivity_state |  |
| application rule name | rule.name | - |  |
| threat prevention rule name | rule.name | - |  |
| voip log type | - | checkpoint.voip_log_type |  |
| dlp rule name | rule.name | - |  |
| email id | - | checkpoint.email_id |  |
| deviceCustomString2 | category | - | checkpoint.category |  |
| email subject | - | checkpoint.email_subject |  |
| sensor mode | - | checkpoint.sensor_mode |  |
| protection id | - | checkpoint.protection_id |  |
| scan invoke type | - | checkpoint.integrity_av_invoke_type |  |
| update status | - | checkpoint.update_status |  |
| peer gateway | - | checkpoint.peer_gateway |  |
| categories | rule.category | - |  |
| deviceCustomString6 | application name | network.application | - |  |
| virus name | - | checkpoint.virus_name |  |
| malware name | - | checkpoint.spyware_name |  |
| malware family | - | checkpoint.malware_family |  |
| deviceCustomString3 | user group | group.name | - |  |
| incident extension | - | checkpoint.incident_extension |  |
| protection type | - | checkpoint.protection_type |  |
| email spool id | - | checkpoint.email_spool_id |  |
| identity type | - | checkpoint.identity_type |  |
| deviceCustomString4 | malware status | - | checkpoint.spyware_status |  |
| threat prevention rule id | rule.id | - |  |
| scan result | - | checkpoint.scan_result |  |
| tcp flags | - | checkpoint.tcp_flags |  |
| destination os | os.name | - |  |
| protection name | - | checkpoint.protection_name |  |
| email control | - | checkpoint.email_control |  |
| frequency | - | checkpoint.frequency |  |
| user response | - | checkpoint.user_status |  |
| deviceCustomString5 | matched category | rule.category | - |  |
| vlan id | network.vlan.id | - |  |
| authentication method | - | checkpoint.auth_method |  |
| email session id | - | checkpoint.email_session_id |  |
| deviceCustomDate2 | subscription expiration | - | checkpoint.subs_exp |  |
| deviceFlexNumber1 | confidence | - | checkpoint.confidence_level |  |
| deviceFlexNumber2 | performance impact | - | checkpoint.performance_impact |  |
| destination phone number | - | checkpoint.dst_phone_number |  |
| flexString1 | application signature id | - | checkpoint.app_sig_id |  |
| flexString2 | malware action | rule.description | - |  |
| attack information | event.action | - |  |
| rule_uid | - | rule.uuid | - |  |
| ifname | - | observer.ingress.interface.name | - |  |
| inzone | - | observer.ingress.zone | - |  |
| outzone | - | observer.egress.zone | - |  |
| product | - | observer.product | - |  |


## Fields [_fields_9]

For a description of each field in the module, see the [exported fields](/reference/filebeat/exported-fields-cef.md) section.
