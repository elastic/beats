---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-suricata.html
---

# Suricata fields [exported-fields-suricata]

Module for handling the EVE JSON logs produced by Suricata.


## suricata [_suricata]

Fields from the Suricata EVE log file.


## eve [_eve]

Fields exported by the EVE JSON logs

**`suricata.eve.event_type`**
:   type: keyword


**`suricata.eve.app_proto_orig`**
:   type: keyword


**`suricata.eve.tcp.tcp_flags`**
:   type: keyword


**`suricata.eve.tcp.psh`**
:   type: boolean


**`suricata.eve.tcp.tcp_flags_tc`**
:   type: keyword


**`suricata.eve.tcp.ack`**
:   type: boolean


**`suricata.eve.tcp.syn`**
:   type: boolean


**`suricata.eve.tcp.state`**
:   type: keyword


**`suricata.eve.tcp.tcp_flags_ts`**
:   type: keyword


**`suricata.eve.tcp.rst`**
:   type: boolean


**`suricata.eve.tcp.fin`**
:   type: boolean


**`suricata.eve.fileinfo.sha1`**
:   type: keyword


**`suricata.eve.fileinfo.tx_id`**
:   type: long


**`suricata.eve.fileinfo.state`**
:   type: keyword


**`suricata.eve.fileinfo.stored`**
:   type: boolean


**`suricata.eve.fileinfo.gaps`**
:   type: boolean


**`suricata.eve.fileinfo.sha256`**
:   type: keyword


**`suricata.eve.fileinfo.md5`**
:   type: keyword


**`suricata.eve.icmp_type`**
:   type: long


**`suricata.eve.pcap_cnt`**
:   type: long


**`suricata.eve.dns.type`**
:   type: keyword


**`suricata.eve.dns.rrtype`**
:   type: keyword


**`suricata.eve.dns.rrname`**
:   type: keyword


**`suricata.eve.dns.rdata`**
:   type: keyword


**`suricata.eve.dns.tx_id`**
:   type: long


**`suricata.eve.dns.ttl`**
:   type: long


**`suricata.eve.dns.rcode`**
:   type: keyword


**`suricata.eve.dns.id`**
:   type: long


**`suricata.eve.flow_id`**
:   type: keyword


**`suricata.eve.email.status`**
:   type: keyword


**`suricata.eve.icmp_code`**
:   type: long


**`suricata.eve.http.redirect`**
:   type: keyword


**`suricata.eve.http.protocol`**
:   type: keyword


**`suricata.eve.http.http_content_type`**
:   type: keyword


**`suricata.eve.in_iface`**
:   type: keyword


**`suricata.eve.alert.metadata`**
:   Metadata about the alert.

type: flattened


**`suricata.eve.alert.category`**
:   type: keyword


**`suricata.eve.alert.rev`**
:   type: long


**`suricata.eve.alert.gid`**
:   type: long


**`suricata.eve.alert.signature`**
:   type: keyword


**`suricata.eve.alert.signature_id`**
:   type: long


**`suricata.eve.alert.protocols`**
:   type: keyword


**`suricata.eve.alert.attack_target`**
:   type: keyword


**`suricata.eve.alert.capec_id`**
:   type: keyword


**`suricata.eve.alert.cwe_id`**
:   type: keyword


**`suricata.eve.alert.malware`**
:   type: keyword


**`suricata.eve.alert.cve`**
:   type: keyword


**`suricata.eve.alert.cvss_v2_base`**
:   type: keyword


**`suricata.eve.alert.cvss_v2_temporal`**
:   type: keyword


**`suricata.eve.alert.cvss_v3_base`**
:   type: keyword


**`suricata.eve.alert.cvss_v3_temporal`**
:   type: keyword


**`suricata.eve.alert.priority`**
:   type: keyword


**`suricata.eve.alert.hostile`**
:   type: keyword


**`suricata.eve.alert.infected`**
:   type: keyword


**`suricata.eve.alert.created_at`**
:   type: date


**`suricata.eve.alert.updated_at`**
:   type: date


**`suricata.eve.alert.classtype`**
:   type: keyword


**`suricata.eve.alert.rule_source`**
:   type: keyword


**`suricata.eve.alert.sid`**
:   type: keyword


**`suricata.eve.alert.affected_product`**
:   type: keyword


**`suricata.eve.alert.deployment`**
:   type: keyword


**`suricata.eve.alert.former_category`**
:   type: keyword


**`suricata.eve.alert.mitre_tool_id`**
:   type: keyword


**`suricata.eve.alert.performance_impact`**
:   type: keyword


**`suricata.eve.alert.signature_severity`**
:   type: keyword


**`suricata.eve.alert.tag`**
:   type: keyword


**`suricata.eve.ssh.client.proto_version`**
:   type: keyword


**`suricata.eve.ssh.client.software_version`**
:   type: keyword


**`suricata.eve.ssh.server.proto_version`**
:   type: keyword


**`suricata.eve.ssh.server.software_version`**
:   type: keyword


**`suricata.eve.stats.capture.kernel_packets`**
:   type: long


**`suricata.eve.stats.capture.kernel_drops`**
:   type: long


**`suricata.eve.stats.capture.kernel_ifdrops`**
:   type: long


**`suricata.eve.stats.uptime`**
:   type: long


**`suricata.eve.stats.detect.alert`**
:   type: long


**`suricata.eve.stats.http.memcap`**
:   type: long


**`suricata.eve.stats.http.memuse`**
:   type: long


**`suricata.eve.stats.file_store.open_files`**
:   type: long


**`suricata.eve.stats.defrag.max_frag_hits`**
:   type: long


**`suricata.eve.stats.defrag.ipv4.timeouts`**
:   type: long


**`suricata.eve.stats.defrag.ipv4.fragments`**
:   type: long


**`suricata.eve.stats.defrag.ipv4.reassembled`**
:   type: long


**`suricata.eve.stats.defrag.ipv6.timeouts`**
:   type: long


**`suricata.eve.stats.defrag.ipv6.fragments`**
:   type: long


**`suricata.eve.stats.defrag.ipv6.reassembled`**
:   type: long


**`suricata.eve.stats.flow.tcp_reuse`**
:   type: long


**`suricata.eve.stats.flow.udp`**
:   type: long


**`suricata.eve.stats.flow.memcap`**
:   type: long


**`suricata.eve.stats.flow.emerg_mode_entered`**
:   type: long


**`suricata.eve.stats.flow.emerg_mode_over`**
:   type: long


**`suricata.eve.stats.flow.tcp`**
:   type: long


**`suricata.eve.stats.flow.icmpv6`**
:   type: long


**`suricata.eve.stats.flow.icmpv4`**
:   type: long


**`suricata.eve.stats.flow.spare`**
:   type: long


**`suricata.eve.stats.flow.memuse`**
:   type: long


**`suricata.eve.stats.tcp.pseudo_failed`**
:   type: long


**`suricata.eve.stats.tcp.ssn_memcap_drop`**
:   type: long


**`suricata.eve.stats.tcp.insert_data_overlap_fail`**
:   type: long


**`suricata.eve.stats.tcp.sessions`**
:   type: long


**`suricata.eve.stats.tcp.pseudo`**
:   type: long


**`suricata.eve.stats.tcp.synack`**
:   type: long


**`suricata.eve.stats.tcp.insert_data_normal_fail`**
:   type: long


**`suricata.eve.stats.tcp.syn`**
:   type: long


**`suricata.eve.stats.tcp.memuse`**
:   type: long


**`suricata.eve.stats.tcp.invalid_checksum`**
:   type: long


**`suricata.eve.stats.tcp.segment_memcap_drop`**
:   type: long


**`suricata.eve.stats.tcp.overlap`**
:   type: long


**`suricata.eve.stats.tcp.insert_list_fail`**
:   type: long


**`suricata.eve.stats.tcp.rst`**
:   type: long


**`suricata.eve.stats.tcp.stream_depth_reached`**
:   type: long


**`suricata.eve.stats.tcp.reassembly_memuse`**
:   type: long


**`suricata.eve.stats.tcp.reassembly_gap`**
:   type: long


**`suricata.eve.stats.tcp.overlap_diff_data`**
:   type: long


**`suricata.eve.stats.tcp.no_flow`**
:   type: long


**`suricata.eve.stats.decoder.avg_pkt_size`**
:   type: long


**`suricata.eve.stats.decoder.bytes`**
:   type: long


**`suricata.eve.stats.decoder.tcp`**
:   type: long


**`suricata.eve.stats.decoder.raw`**
:   type: long


**`suricata.eve.stats.decoder.ppp`**
:   type: long


**`suricata.eve.stats.decoder.vlan_qinq`**
:   type: long


**`suricata.eve.stats.decoder.null`**
:   type: long


**`suricata.eve.stats.decoder.ltnull.unsupported_type`**
:   type: long


**`suricata.eve.stats.decoder.ltnull.pkt_too_small`**
:   type: long


**`suricata.eve.stats.decoder.invalid`**
:   type: long


**`suricata.eve.stats.decoder.gre`**
:   type: long


**`suricata.eve.stats.decoder.ipv4`**
:   type: long


**`suricata.eve.stats.decoder.ipv6`**
:   type: long


**`suricata.eve.stats.decoder.pkts`**
:   type: long


**`suricata.eve.stats.decoder.ipv6_in_ipv6`**
:   type: long


**`suricata.eve.stats.decoder.ipraw.invalid_ip_version`**
:   type: long


**`suricata.eve.stats.decoder.pppoe`**
:   type: long


**`suricata.eve.stats.decoder.udp`**
:   type: long


**`suricata.eve.stats.decoder.dce.pkt_too_small`**
:   type: long


**`suricata.eve.stats.decoder.vlan`**
:   type: long


**`suricata.eve.stats.decoder.sctp`**
:   type: long


**`suricata.eve.stats.decoder.max_pkt_size`**
:   type: long


**`suricata.eve.stats.decoder.teredo`**
:   type: long


**`suricata.eve.stats.decoder.mpls`**
:   type: long


**`suricata.eve.stats.decoder.sll`**
:   type: long


**`suricata.eve.stats.decoder.icmpv6`**
:   type: long


**`suricata.eve.stats.decoder.icmpv4`**
:   type: long


**`suricata.eve.stats.decoder.erspan`**
:   type: long


**`suricata.eve.stats.decoder.ethernet`**
:   type: long


**`suricata.eve.stats.decoder.ipv4_in_ipv6`**
:   type: long


**`suricata.eve.stats.decoder.ieee8021ah`**
:   type: long


**`suricata.eve.stats.dns.memcap_global`**
:   type: long


**`suricata.eve.stats.dns.memcap_state`**
:   type: long


**`suricata.eve.stats.dns.memuse`**
:   type: long


**`suricata.eve.stats.flow_mgr.rows_busy`**
:   type: long


**`suricata.eve.stats.flow_mgr.flows_timeout`**
:   type: long


**`suricata.eve.stats.flow_mgr.flows_notimeout`**
:   type: long


**`suricata.eve.stats.flow_mgr.rows_skipped`**
:   type: long


**`suricata.eve.stats.flow_mgr.closed_pruned`**
:   type: long


**`suricata.eve.stats.flow_mgr.new_pruned`**
:   type: long


**`suricata.eve.stats.flow_mgr.flows_removed`**
:   type: long


**`suricata.eve.stats.flow_mgr.bypassed_pruned`**
:   type: long


**`suricata.eve.stats.flow_mgr.est_pruned`**
:   type: long


**`suricata.eve.stats.flow_mgr.flows_timeout_inuse`**
:   type: long


**`suricata.eve.stats.flow_mgr.flows_checked`**
:   type: long


**`suricata.eve.stats.flow_mgr.rows_maxlen`**
:   type: long


**`suricata.eve.stats.flow_mgr.rows_checked`**
:   type: long


**`suricata.eve.stats.flow_mgr.rows_empty`**
:   type: long


**`suricata.eve.stats.app_layer.flow.tls`**
:   type: long


**`suricata.eve.stats.app_layer.flow.ftp`**
:   type: long


**`suricata.eve.stats.app_layer.flow.http`**
:   type: long


**`suricata.eve.stats.app_layer.flow.failed_udp`**
:   type: long


**`suricata.eve.stats.app_layer.flow.dns_udp`**
:   type: long


**`suricata.eve.stats.app_layer.flow.dns_tcp`**
:   type: long


**`suricata.eve.stats.app_layer.flow.smtp`**
:   type: long


**`suricata.eve.stats.app_layer.flow.failed_tcp`**
:   type: long


**`suricata.eve.stats.app_layer.flow.msn`**
:   type: long


**`suricata.eve.stats.app_layer.flow.ssh`**
:   type: long


**`suricata.eve.stats.app_layer.flow.imap`**
:   type: long


**`suricata.eve.stats.app_layer.flow.dcerpc_udp`**
:   type: long


**`suricata.eve.stats.app_layer.flow.dcerpc_tcp`**
:   type: long


**`suricata.eve.stats.app_layer.flow.smb`**
:   type: long


**`suricata.eve.stats.app_layer.tx.tls`**
:   type: long


**`suricata.eve.stats.app_layer.tx.ftp`**
:   type: long


**`suricata.eve.stats.app_layer.tx.http`**
:   type: long


**`suricata.eve.stats.app_layer.tx.dns_udp`**
:   type: long


**`suricata.eve.stats.app_layer.tx.dns_tcp`**
:   type: long


**`suricata.eve.stats.app_layer.tx.smtp`**
:   type: long


**`suricata.eve.stats.app_layer.tx.ssh`**
:   type: long


**`suricata.eve.stats.app_layer.tx.dcerpc_udp`**
:   type: long


**`suricata.eve.stats.app_layer.tx.dcerpc_tcp`**
:   type: long


**`suricata.eve.stats.app_layer.tx.smb`**
:   type: long


**`suricata.eve.tls.notbefore`**
:   type: date


**`suricata.eve.tls.issuerdn`**
:   type: keyword


**`suricata.eve.tls.sni`**
:   type: keyword


**`suricata.eve.tls.version`**
:   type: keyword


**`suricata.eve.tls.session_resumed`**
:   type: boolean


**`suricata.eve.tls.fingerprint`**
:   type: keyword


**`suricata.eve.tls.serial`**
:   type: keyword


**`suricata.eve.tls.notafter`**
:   type: date


**`suricata.eve.tls.subject`**
:   type: keyword


**`suricata.eve.tls.ja3s.string`**
:   type: keyword


**`suricata.eve.tls.ja3s.hash`**
:   type: keyword


**`suricata.eve.tls.ja3.string`**
:   type: keyword


**`suricata.eve.tls.ja3.hash`**
:   type: keyword


**`suricata.eve.app_proto_ts`**
:   type: keyword


**`suricata.eve.flow.age`**
:   type: long


**`suricata.eve.flow.state`**
:   type: keyword


**`suricata.eve.flow.reason`**
:   type: keyword


**`suricata.eve.flow.alerted`**
:   type: boolean


**`suricata.eve.tx_id`**
:   type: long


**`suricata.eve.app_proto_tc`**
:   type: keyword


**`suricata.eve.smtp.rcpt_to`**
:   type: keyword


**`suricata.eve.smtp.mail_from`**
:   type: keyword


**`suricata.eve.smtp.helo`**
:   type: keyword


**`suricata.eve.app_proto_expected`**
:   type: keyword


