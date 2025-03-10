---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-cef-module.html
---

# CEF fields [exported-fields-cef-module]

Module for receiving CEF logs over Syslog. The module adds vendor specific fields in addition to the fields the decode_cef processor provides.


## forcepoint [_forcepoint]

Fields for Forcepoint Custom String mappings

**`forcepoint.virus_id`**
:   Virus ID

type: keyword



## checkpoint [_checkpoint]

Fields for Check Point custom string mappings.

**`checkpoint.app_risk`**
:   Application risk.

type: keyword


**`checkpoint.app_severity`**
:   Application threat severity.

type: keyword


**`checkpoint.app_sig_id`**
:   The signature ID which the application was detected by.

type: keyword


**`checkpoint.auth_method`**
:   Password authentication protocol used.

type: keyword


**`checkpoint.category`**
:   Category.

type: keyword


**`checkpoint.confidence_level`**
:   Confidence level determined.

type: integer


**`checkpoint.connectivity_state`**
:   Connectivity state.

type: keyword


**`checkpoint.cookie`**
:   IKE cookie.

type: keyword


**`checkpoint.dst_phone_number`**
:   Destination IP-Phone.

type: keyword


**`checkpoint.email_control`**
:   Engine name.

type: keyword


**`checkpoint.email_id`**
:   Internal email ID.

type: keyword


**`checkpoint.email_recipients_num`**
:   Number of recipients.

type: long


**`checkpoint.email_session_id`**
:   Internal email session ID.

type: keyword


**`checkpoint.email_spool_id`**
:   Internal email spool ID.

type: keyword


**`checkpoint.email_subject`**
:   Email subject.

type: keyword


**`checkpoint.event_count`**
:   Number of events associated with the log.

type: long


**`checkpoint.frequency`**
:   Scan frequency.

type: keyword


**`checkpoint.icmp_type`**
:   ICMP type.

type: long


**`checkpoint.icmp_code`**
:   ICMP code.

type: long


**`checkpoint.identity_type`**
:   Identity type.

type: keyword


**`checkpoint.incident_extension`**
:   Format of original data.

type: keyword


**`checkpoint.integrity_av_invoke_type`**
:   Scan invoke type.

type: keyword


**`checkpoint.malware_family`**
:   Malware family.

type: keyword


**`checkpoint.peer_gateway`**
:   Main IP of the peer Security Gateway.

type: ip


**`checkpoint.performance_impact`**
:   Protection performance impact.

type: integer


**`checkpoint.protection_id`**
:   Protection malware ID.

type: keyword


**`checkpoint.protection_name`**
:   Specific signature name of the attack.

type: keyword


**`checkpoint.protection_type`**
:   Type of protection used to detect the attack.

type: keyword


**`checkpoint.scan_result`**
:   Scan result.

type: keyword


**`checkpoint.sensor_mode`**
:   Sensor mode.

type: keyword


**`checkpoint.severity`**
:   Threat severity.

type: keyword


**`checkpoint.spyware_name`**
:   Spyware name.

type: keyword


**`checkpoint.spyware_status`**
:   Spyware status.

type: keyword


**`checkpoint.subs_exp`**
:   The expiration date of the subscription.

type: date


**`checkpoint.tcp_flags`**
:   TCP packet flags.

type: keyword


**`checkpoint.termination_reason`**
:   Termination reason.

type: keyword


**`checkpoint.update_status`**
:   Update status.

type: keyword


**`checkpoint.user_status`**
:   User response.

type: keyword


**`checkpoint.uuid`**
:   External ID.

type: keyword


**`checkpoint.virus_name`**
:   Virus name.

type: keyword


**`checkpoint.voip_log_type`**
:   VoIP log types.

type: keyword



## cef.extensions [_cef_extensions]

Extra vendor-specific extensions.

**`cef.extensions.cp_app_risk`**
:   type: keyword


**`cef.extensions.cp_severity`**
:   type: keyword


**`cef.extensions.ifname`**
:   type: keyword


**`cef.extensions.inzone`**
:   type: keyword


**`cef.extensions.layer_uuid`**
:   type: keyword


**`cef.extensions.layer_name`**
:   type: keyword


**`cef.extensions.logid`**
:   type: keyword


**`cef.extensions.loguid`**
:   type: keyword


**`cef.extensions.match_id`**
:   type: keyword


**`cef.extensions.nat_addtnl_rulenum`**
:   type: keyword


**`cef.extensions.nat_rulenum`**
:   type: keyword


**`cef.extensions.origin`**
:   type: keyword


**`cef.extensions.originsicname`**
:   type: keyword


**`cef.extensions.outzone`**
:   type: keyword


**`cef.extensions.parent_rule`**
:   type: keyword


**`cef.extensions.product`**
:   type: keyword


**`cef.extensions.rule_action`**
:   type: keyword


**`cef.extensions.rule_uid`**
:   type: keyword


**`cef.extensions.sequencenum`**
:   type: keyword


**`cef.extensions.service_id`**
:   type: keyword


**`cef.extensions.version`**
:   type: keyword


