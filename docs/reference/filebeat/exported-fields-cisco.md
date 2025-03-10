---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-cisco.html
---

# Cisco fields [exported-fields-cisco]

Module for handling Cisco network device logs.


## cisco.amp [_cisco_amp]

Module for parsing Cisco AMP logs.

**`cisco.amp.timestamp_nanoseconds`**
:   The timestamp in Epoch nanoseconds.

type: date


**`cisco.amp.event_type_id`**
:   A sub ID of the event, depending on event type.

type: keyword


**`cisco.amp.detection`**
:   The name of the malware detected.

type: keyword


**`cisco.amp.detection_id`**
:   The ID of the detection.

type: keyword


**`cisco.amp.connector_guid`**
:   The GUID of the connector sending information to AMP.

type: keyword


**`cisco.amp.group_guids`**
:   An array of group GUIDS related to the connector sending information to AMP.

type: keyword


**`cisco.amp.vulnerabilities`**
:   An array of related vulnerabilities to the malicious event.

type: flattened


**`cisco.amp.scan.description`**
:   Description of an event related to a scan being initiated, for example the specific directory name.

type: keyword


**`cisco.amp.scan.clean`**
:   Boolean value if a scanned file was clean or not.

type: boolean


**`cisco.amp.scan.scanned_files`**
:   Count of files scanned in a directory.

type: long


**`cisco.amp.scan.scanned_processes`**
:   Count of processes scanned related to a single scan event.

type: long


**`cisco.amp.scan.scanned_paths`**
:   Count of different directories scanned related to a single scan event.

type: long


**`cisco.amp.scan.malicious_detections`**
:   Count of malicious files or documents detected related to a single scan event.

type: long


**`cisco.amp.computer.connector_guid`**
:   The GUID of the connector, similar to top level connector_guid, but unique if multiple connectors are involved.

type: keyword


**`cisco.amp.computer.external_ip`**
:   The external IP of the related host.

type: ip


**`cisco.amp.computer.active`**
:   If the current endpoint is active or not.

type: boolean


**`cisco.amp.computer.network_addresses`**
:   All network interface information on the related host.

type: flattened


**`cisco.amp.file.disposition`**
:   Categorization of file, for example "Malicious" or "Clean".

type: keyword


**`cisco.amp.network_info.disposition`**
:   Categorization of a network event related to a file, for example "Malicious" or "Clean".

type: keyword


**`cisco.amp.network_info.nfm.direction`**
:   The current direction based on source and destination IP.

type: keyword


**`cisco.amp.related.mac`**
:   An array of all related MAC addresses.

type: keyword


**`cisco.amp.related.cve`**
:   An array of all related MAC addresses.

type: keyword


**`cisco.amp.cloud_ioc.description`**
:   Description of the related IOC for specific IOC events from AMP.

type: keyword


**`cisco.amp.cloud_ioc.short_description`**
:   Short description of the related IOC for specific IOC events from AMP.

type: keyword


**`cisco.amp.network_info.parent.disposition`**
:   Categorization of a IOC for example "Malicious" or "Clean".

type: keyword


**`cisco.amp.network_info.parent.identity.md5`**
:   MD5 hash of the related IOC.

type: keyword


**`cisco.amp.network_info.parent.identity.sha1`**
:   SHA1 hash of the related IOC.

type: keyword


**`cisco.amp.network_info.parent.identify.sha256`**
:   SHA256 hash of the related IOC.

type: keyword


**`cisco.amp.file.archived_file.disposition`**
:   Categorization of a file archive related to a file, for example "Malicious" or "Clean".

type: keyword


**`cisco.amp.file.archived_file.identity.md5`**
:   MD5 hash of the archived file related to the malicious event.

type: keyword


**`cisco.amp.file.archived_file.identity.sha1`**
:   SHA1 hash of the archived file related to the malicious event.

type: keyword


**`cisco.amp.file.archived_file.identity.sha256`**
:   SHA256 hash of the archived file related to the malicious event.

type: keyword


**`cisco.amp.file.attack_details.application`**
:   The application name related to Exploit Prevention events.

type: keyword


**`cisco.amp.file.attack_details.attacked_module`**
:   Path to the executable or dll that was attacked and detected by Exploit Prevention.

type: keyword


**`cisco.amp.file.attack_details.base_address`**
:   The base memory address related to the exploit detected.

type: keyword


**`cisco.amp.file.attack_details.suspicious_files`**
:   An array of related files when an attack is detected by Exploit Prevention.

type: keyword


**`cisco.amp.file.parent.disposition`**
:   Categorization of parrent, for example "Malicious" or "Clean".

type: keyword


**`cisco.amp.error.description`**
:   Description of an endpoint error event.

type: keyword


**`cisco.amp.error.error_code`**
:   The error code describing the related error event.

type: keyword


**`cisco.amp.threat_hunting.severity`**
:   Severity result of the threat hunt registered to the malicious event. Can be Low-Critical.

type: keyword


**`cisco.amp.threat_hunting.incident_report_guid`**
:   The GUID of the related threat hunting report.

type: keyword


**`cisco.amp.threat_hunting.incident_hunt_guid`**
:   The GUID of the related investigation tracking issue.

type: keyword


**`cisco.amp.threat_hunting.incident_title`**
:   Title of the incident related to the threat hunting activity.

type: keyword


**`cisco.amp.threat_hunting.incident_summary`**
:   Summary of the outcome on the threat hunting activity.

type: keyword


**`cisco.amp.threat_hunting.incident_remediation`**
:   Recommendations to resolve the vulnerability or exploited host.

type: keyword


**`cisco.amp.threat_hunting.incident_id`**
:   The id of the related incident for the threat hunting activity.

type: keyword


**`cisco.amp.threat_hunting.incident_end_time`**
:   When the threat hunt finalized or closed.

type: date


**`cisco.amp.threat_hunting.incident_start_time`**
:   When the threat hunt was initiated.

type: date


**`cisco.amp.file.attack_details.indicators`**
:   Different indicator types that matches the exploit detected, for example different MITRE tactics.

type: flattened


**`cisco.amp.threat_hunting.tactics`**
:   List of all MITRE tactics related to the incident found.

type: flattened


**`cisco.amp.threat_hunting.techniques`**
:   List of all MITRE techniques related to the incident found.

type: flattened


**`cisco.amp.tactics`**
:   List of all MITRE tactics related to the incident found.

type: flattened


**`cisco.amp.mitre_tactics`**
:   Array of all related mitre tactic ID’s

type: keyword


**`cisco.amp.techniques`**
:   List of all MITRE techniques related to the incident found.

type: flattened


**`cisco.amp.mitre_techniques`**
:   Array of all related mitre technique ID’s

type: keyword


**`cisco.amp.command_line.arguments`**
:   The CLI arguments related to the Cloud Threat IOC reported by Cisco.

type: keyword


**`cisco.amp.bp_data`**
:   Endpoint isolation information

type: flattened



## cisco.asa [_cisco_asa]

Fields for Cisco ASA Firewall.

**`cisco.asa.message_id`**
:   The Cisco ASA message identifier.

type: keyword


**`cisco.asa.suffix`**
:   Optional suffix after %ASA identifier.

type: keyword

example: session


**`cisco.asa.source_interface`**
:   Source interface for the flow or event.

type: keyword


**`cisco.asa.destination_interface`**
:   Destination interface for the flow or event.

type: keyword


**`cisco.asa.rule_name`**
:   Name of the Access Control List rule that matched this event.

type: keyword


**`cisco.asa.source_username`**
:   Name of the user that is the source for this event.

type: keyword


**`cisco.asa.source_user_security_group_tag`**
:   The Security Group Tag for the source user. Security Group Tag are 16-bit identifiers used to represent logical group privilege.

type: long


**`cisco.asa.destination_username`**
:   Name of the user that is the destination for this event.

type: keyword


**`cisco.asa.destination_user_security_group_tag`**
:   The Security Group Tag for the destination user. Security Group Tag are 16-bit identifiers used to represent logical group privilege.

type: long


**`cisco.asa.mapped_source_ip`**
:   The translated source IP address.

type: ip


**`cisco.asa.mapped_source_host`**
:   The translated source host.

type: keyword


**`cisco.asa.mapped_source_port`**
:   The translated source port.

type: long


**`cisco.asa.mapped_destination_ip`**
:   The translated destination IP address.

type: ip


**`cisco.asa.mapped_destination_host`**
:   The translated destination host.

type: keyword


**`cisco.asa.mapped_destination_port`**
:   The translated destination port.

type: long


**`cisco.asa.threat_level`**
:   Threat level for malware / botnet traffic. One of very-low, low, moderate, high or very-high.

type: keyword


**`cisco.asa.threat_category`**
:   Category for the malware / botnet traffic. For example: virus, botnet, trojan, etc.

type: keyword


**`cisco.asa.connection_id`**
:   Unique identifier for a flow.

type: keyword


**`cisco.asa.icmp_type`**
:   ICMP type.

type: short


**`cisco.asa.icmp_code`**
:   ICMP code.

type: short


**`cisco.asa.connection_type`**
:   The VPN connection type

type: keyword


**`cisco.asa.dap_records`**
:   The assigned DAP records

type: keyword


**`cisco.asa.command_line_arguments`**
:   The command line arguments logged by the local audit log

type: keyword


**`cisco.asa.assigned_ip`**
:   The IP address assigned to a VPN client successfully connecting

type: ip


**`cisco.asa.privilege.old`**
:   When a users privilege is changed this is the old value

type: keyword


**`cisco.asa.privilege.new`**
:   When a users privilege is changed this is the new value

type: keyword


**`cisco.asa.burst.object`**
:   The related object for burst warnings

type: keyword


**`cisco.asa.burst.id`**
:   The related rate ID for burst warnings

type: keyword


**`cisco.asa.burst.current_rate`**
:   The current burst rate seen

type: keyword


**`cisco.asa.burst.configured_rate`**
:   The current configured burst rate

type: keyword


**`cisco.asa.burst.avg_rate`**
:   The current average burst rate seen

type: keyword


**`cisco.asa.burst.configured_avg_rate`**
:   The current configured average burst rate allowed

type: keyword


**`cisco.asa.burst.cumulative_count`**
:   The total count of burst rate hits since the object was created or cleared

type: keyword


**`cisco.asa.termination_user`**
:   AAA name of user requesting termination

type: keyword


**`cisco.asa.webvpn.group_name`**
:   The WebVPN group name the user belongs to

type: keyword


**`cisco.asa.termination_initiator`**
:   Interface name of the side that initiated the teardown

type: keyword


**`cisco.asa.tunnel_type`**
:   SA type (remote access or L2L)

type: keyword


**`cisco.asa.session_type`**
:   Session type (for example, IPsec or UDP)

type: keyword



## cisco.ftd [_cisco_ftd]

Fields for Cisco Firepower Threat Defense Firewall.

**`cisco.ftd.message_id`**
:   The Cisco FTD message identifier.

type: keyword


**`cisco.ftd.suffix`**
:   Optional suffix after %FTD identifier.

type: keyword

example: session


**`cisco.ftd.source_interface`**
:   Source interface for the flow or event.

type: keyword


**`cisco.ftd.destination_interface`**
:   Destination interface for the flow or event.

type: keyword


**`cisco.ftd.rule_name`**
:   Name of the Access Control List rule that matched this event.

type: keyword


**`cisco.ftd.source_username`**
:   Name of the user that is the source for this event.

type: keyword


**`cisco.ftd.destination_username`**
:   Name of the user that is the destination for this event.

type: keyword


**`cisco.ftd.mapped_source_ip`**
:   The translated source IP address. Use ECS source.nat.ip.

type: ip


**`cisco.ftd.mapped_source_host`**
:   The translated source host.

type: keyword


**`cisco.ftd.mapped_source_port`**
:   The translated source port. Use ECS source.nat.port.

type: long


**`cisco.ftd.mapped_destination_ip`**
:   The translated destination IP address. Use ECS destination.nat.ip.

type: ip


**`cisco.ftd.mapped_destination_host`**
:   The translated destination host.

type: keyword


**`cisco.ftd.mapped_destination_port`**
:   The translated destination port. Use ECS destination.nat.port.

type: long


**`cisco.ftd.threat_level`**
:   Threat level for malware / botnet traffic. One of very-low, low, moderate, high or very-high.

type: keyword


**`cisco.ftd.threat_category`**
:   Category for the malware / botnet traffic. For example: virus, botnet, trojan, etc.

type: keyword


**`cisco.ftd.connection_id`**
:   Unique identifier for a flow.

type: keyword


**`cisco.ftd.icmp_type`**
:   ICMP type.

type: short


**`cisco.ftd.icmp_code`**
:   ICMP code.

type: short


**`cisco.ftd.security`**
:   Raw fields for Security Events.

type: object


**`cisco.ftd.connection_type`**
:   The VPN connection type

type: keyword


**`cisco.ftd.dap_records`**
:   The assigned DAP records

type: keyword


**`cisco.ftd.termination_user`**
:   AAA name of user requesting termination

type: keyword


**`cisco.ftd.webvpn.group_name`**
:   The WebVPN group name the user belongs to

type: keyword


**`cisco.ftd.termination_initiator`**
:   Interface name of the side that initiated the teardown

type: keyword



## cisco.ios [_cisco_ios]

Fields for Cisco IOS logs.

**`cisco.ios.access_list`**
:   Name of the IP access list.

type: keyword


**`cisco.ios.facility`**
:   The facility to which the message refers (for example, SNMP, SYS, and so forth). A facility can be a hardware device, a protocol, or a module of the system software. It denotes the source or the cause of the system message.

type: keyword

example: SEC



## cisco.umbrella [_cisco_umbrella]

Fields for Cisco Umbrella.

**`cisco.umbrella.identities`**
:   An array of the different identities related to the event.

type: keyword


**`cisco.umbrella.categories`**
:   The security or content categories that the destination matches.

type: keyword


**`cisco.umbrella.policy_identity_type`**
:   The first identity type matched with this request. Available in version 3 and above.

type: keyword


**`cisco.umbrella.identity_types`**
:   The type of identity that made the request. For example, Roaming Computer or Network.

type: keyword


**`cisco.umbrella.blocked_categories`**
:   The categories that resulted in the destination being blocked. Available in version 4 and above.

type: keyword


**`cisco.umbrella.content_type`**
:   The type of web content, typically text/html.

type: keyword


**`cisco.umbrella.sha_sha256`**
:   Hex digest of the response content.

type: keyword


**`cisco.umbrella.av_detections`**
:   The detection name according to the antivirus engine used in file inspection.

type: keyword


**`cisco.umbrella.puas`**
:   A list of all potentially unwanted application (PUA) results for the proxied file as returned by the antivirus scanner.

type: keyword


**`cisco.umbrella.amp_disposition`**
:   The status of the files proxied and scanned by Cisco Advanced Malware Protection (AMP) as part of the Umbrella File Inspection feature; can be Clean, Malicious or Unknown.

type: keyword


**`cisco.umbrella.amp_malware_name`**
:   If Malicious, the name of the malware according to AMP.

type: keyword


**`cisco.umbrella.amp_score`**
:   The score of the malware from AMP. This field is not currently used and will be blank.

type: keyword


**`cisco.umbrella.datacenter`**
:   The name of the Umbrella Data Center that processed the user-generated traffic.

type: keyword


**`cisco.umbrella.origin_id`**
:   The unique identity of the network tunnel.

type: keyword


