---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-juniper.html
---

# Juniper JUNOS fields [exported-fields-juniper]

juniper fields.


## juniper.srx [_juniper_srx]

Module for parsing junipersrx syslog.

**`juniper.srx.reason`**
:   reason

type: keyword


**`juniper.srx.connection_tag`**
:   connection tag

type: keyword


**`juniper.srx.service_name`**
:   service name

type: keyword


**`juniper.srx.nat_connection_tag`**
:   nat connection tag

type: keyword


**`juniper.srx.src_nat_rule_type`**
:   src nat rule type

type: keyword


**`juniper.srx.src_nat_rule_name`**
:   src nat rule name

type: keyword


**`juniper.srx.dst_nat_rule_type`**
:   dst nat rule type

type: keyword


**`juniper.srx.dst_nat_rule_name`**
:   dst nat rule name

type: keyword


**`juniper.srx.protocol_id`**
:   protocol id

type: keyword


**`juniper.srx.policy_name`**
:   policy name

type: keyword


**`juniper.srx.session_id_32`**
:   session id 32

type: keyword


**`juniper.srx.session_id`**
:   session id

type: keyword


**`juniper.srx.outbound_packets`**
:   packets from client

type: integer


**`juniper.srx.outbound_bytes`**
:   bytes from client

type: integer


**`juniper.srx.inbound_packets`**
:   packets from server

type: integer


**`juniper.srx.inbound_bytes`**
:   bytes from server

type: integer


**`juniper.srx.elapsed_time`**
:   elapsed time

type: date


**`juniper.srx.application`**
:   application

type: keyword


**`juniper.srx.nested_application`**
:   nested application

type: keyword


**`juniper.srx.username`**
:   username

type: keyword


**`juniper.srx.roles`**
:   roles

type: keyword


**`juniper.srx.encrypted`**
:   encrypted

type: keyword


**`juniper.srx.application_category`**
:   application category

type: keyword


**`juniper.srx.application_sub_category`**
:   application sub category

type: keyword


**`juniper.srx.application_characteristics`**
:   application characteristics

type: keyword


**`juniper.srx.secure_web_proxy_session_type`**
:   secure web proxy session type

type: keyword


**`juniper.srx.peer_session_id`**
:   peer session id

type: keyword


**`juniper.srx.peer_source_address`**
:   peer source address

type: ip


**`juniper.srx.peer_source_port`**
:   peer source port

type: integer


**`juniper.srx.peer_destination_address`**
:   peer destination address

type: ip


**`juniper.srx.peer_destination_port`**
:   peer destination port

type: integer


**`juniper.srx.hostname`**
:   hostname

type: keyword


**`juniper.srx.src_vrf_grp`**
:   src_vrf_grp

type: keyword


**`juniper.srx.dst_vrf_grp`**
:   dst_vrf_grp

type: keyword


**`juniper.srx.icmp_type`**
:   icmp type

type: integer


**`juniper.srx.process`**
:   process that generated the message

type: keyword


**`juniper.srx.apbr_rule_type`**
:   apbr rule type

type: keyword


**`juniper.srx.dscp_value`**
:   apbr rule type

type: integer


**`juniper.srx.logical_system_name`**
:   logical system name

type: keyword


**`juniper.srx.profile_name`**
:   profile name

type: keyword


**`juniper.srx.routing_instance`**
:   routing instance

type: keyword


**`juniper.srx.rule_name`**
:   rule name

type: keyword


**`juniper.srx.uplink_tx_bytes`**
:   uplink tx bytes

type: integer


**`juniper.srx.uplink_rx_bytes`**
:   uplink rx bytes

type: integer


**`juniper.srx.obj`**
:   url path

type: keyword


**`juniper.srx.url`**
:   url domain

type: keyword


**`juniper.srx.profile`**
:   filter profile

type: keyword


**`juniper.srx.category`**
:   filter category

type: keyword


**`juniper.srx.filename`**
:   filename

type: keyword


**`juniper.srx.temporary_filename`**
:   temporary_filename

type: keyword


**`juniper.srx.name`**
:   name

type: keyword


**`juniper.srx.error_message`**
:   error_message

type: keyword


**`juniper.srx.error_code`**
:   error_code

type: keyword


**`juniper.srx.action`**
:   action

type: keyword


**`juniper.srx.protocol`**
:   protocol

type: keyword


**`juniper.srx.protocol_name`**
:   protocol name

type: keyword


**`juniper.srx.type`**
:   type

type: keyword


**`juniper.srx.repeat_count`**
:   repeat count

type: integer


**`juniper.srx.alert`**
:   repeat alert

type: keyword


**`juniper.srx.message_type`**
:   message type

type: keyword


**`juniper.srx.threat_severity`**
:   threat severity

type: keyword


**`juniper.srx.application_name`**
:   application name

type: keyword


**`juniper.srx.attack_name`**
:   attack name

type: keyword


**`juniper.srx.index`**
:   index

type: keyword


**`juniper.srx.message`**
:   mesagge

type: keyword


**`juniper.srx.epoch_time`**
:   epoch time

type: date


**`juniper.srx.packet_log_id`**
:   packet log id

type: integer


**`juniper.srx.export_id`**
:   packet log id

type: integer


**`juniper.srx.ddos_application_name`**
:   ddos application name

type: keyword


**`juniper.srx.connection_hit_rate`**
:   connection hit rate

type: integer


**`juniper.srx.time_scope`**
:   time scope

type: keyword


**`juniper.srx.context_hit_rate`**
:   context hit rate

type: integer


**`juniper.srx.context_value_hit_rate`**
:   context value hit rate

type: integer


**`juniper.srx.time_count`**
:   time count

type: integer


**`juniper.srx.time_period`**
:   time period

type: integer


**`juniper.srx.context_value`**
:   context value

type: keyword


**`juniper.srx.context_name`**
:   context name

type: keyword


**`juniper.srx.ruleebase_name`**
:   ruleebase name

type: keyword


**`juniper.srx.verdict_source`**
:   verdict source

type: keyword


**`juniper.srx.verdict_number`**
:   verdict number

type: integer


**`juniper.srx.file_category`**
:   file category

type: keyword


**`juniper.srx.sample_sha256`**
:   sample sha256

type: keyword


**`juniper.srx.malware_info`**
:   malware info

type: keyword


**`juniper.srx.client_ip`**
:   client ip

type: ip


**`juniper.srx.tenant_id`**
:   tenant id

type: keyword


**`juniper.srx.timestamp`**
:   timestamp

type: date


**`juniper.srx.th`**
:   th

type: keyword


**`juniper.srx.status`**
:   status

type: keyword


**`juniper.srx.state`**
:   state

type: keyword


**`juniper.srx.file_hash_lookup`**
:   file hash lookup

type: keyword


**`juniper.srx.file_name`**
:   file name

type: keyword


**`juniper.srx.action_detail`**
:   action detail

type: keyword


**`juniper.srx.sub_category`**
:   sub category

type: keyword


**`juniper.srx.feed_name`**
:   feed name

type: keyword


**`juniper.srx.occur_count`**
:   occur count

type: integer


**`juniper.srx.tag`**
:   system log message tag, which uniquely identifies the message.

type: keyword


