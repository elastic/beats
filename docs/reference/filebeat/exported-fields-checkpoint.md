---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-checkpoint.html
---

# Checkpoint fields [exported-fields-checkpoint]

Some checkpoint module


## checkpoint [_checkpoint_2]

Module for parsing Checkpoint syslog.

**`checkpoint.confidence_level`**
:   Confidence level determined by ThreatCloud.

type: integer


**`checkpoint.calc_desc`**
:   Log description.

type: keyword


**`checkpoint.dst_country`**
:   Destination country.

type: keyword


**`checkpoint.dst_user_name`**
:   Connected user name on the destination IP.

type: keyword


**`checkpoint.email_id`**
:   Email number in smtp connection.

type: keyword


**`checkpoint.email_subject`**
:   Original email subject.

type: keyword


**`checkpoint.email_session_id`**
:   Connection uuid.

type: keyword


**`checkpoint.event_count`**
:   Number of events associated with the log.

type: long


**`checkpoint.sys_message`**
:   System messages

type: keyword


**`checkpoint.logid`**
:   System messages

type: keyword


**`checkpoint.failure_impact`**
:   The impact of update service failure.

type: keyword


**`checkpoint.id`**
:   Override application ID.

type: integer


**`checkpoint.identity_src`**
:   The source for authentication identity information.

type: keyword


**`checkpoint.information`**
:   Policy installation status for a specific blade.

type: keyword


**`checkpoint.layer_name`**
:   Layer name.

type: keyword


**`checkpoint.layer_uuid`**
:   Layer UUID.

type: keyword


**`checkpoint.log_id`**
:   Unique identity for logs.

type: integer


**`checkpoint.malware_family`**
:   Additional information on protection.

type: keyword


**`checkpoint.origin_sic_name`**
:   Machine SIC.

type: keyword


**`checkpoint.policy_mgmt`**
:   Name of the Management Server that manages this Security Gateway.

type: keyword


**`checkpoint.policy_name`**
:   Name of the last policy that this Security Gateway fetched.

type: keyword


**`checkpoint.protection_id`**
:   Protection malware id.

type: keyword


**`checkpoint.protection_name`**
:   Specific signature name of the attack.

type: keyword


**`checkpoint.protection_type`**
:   Type of protection used to detect the attack.

type: keyword


**`checkpoint.protocol`**
:   Protocol detected on the connection.

type: keyword


**`checkpoint.proxy_src_ip`**
:   Sender source IP (even when using proxy).

type: ip


**`checkpoint.rule`**
:   Matched rule number.

type: integer


**`checkpoint.rule_action`**
:   Action of the matched rule in the access policy.

type: keyword


**`checkpoint.scan_direction`**
:   Scan direction.

type: keyword


**`checkpoint.session_id`**
:   Log uuid.

type: keyword


**`checkpoint.source_os`**
:   OS which generated the attack.

type: keyword


**`checkpoint.src_country`**
:   Country name, derived from connection source IP address.

type: keyword


**`checkpoint.src_user_name`**
:   User name connected to source IP

type: keyword


**`checkpoint.ticket_id`**
:   Unique ID per file.

type: keyword


**`checkpoint.tls_server_host_name`**
:   SNI/CN from encrypted TLS connection used by URLF for categorization.

type: keyword


**`checkpoint.verdict`**
:   TE engine verdict Possible values: Malicious/Benign/Error.

type: keyword


**`checkpoint.user`**
:   Source user name.

type: keyword


**`checkpoint.vendor_list`**
:   The vendor name that provided the verdict for a malicious URL.

type: keyword


**`checkpoint.web_server_type`**
:   Web server detected in the HTTP response.

type: keyword


**`checkpoint.client_name`**
:   Client Application or Software Blade that detected the event.

type: keyword


**`checkpoint.client_version`**
:   Build version of SandBlast Agent client installed on the computer.

type: keyword


**`checkpoint.extension_version`**
:   Build version of the SandBlast Agent browser extension.

type: keyword


**`checkpoint.host_time`**
:   Local time on the endpoint computer.

type: keyword


**`checkpoint.installed_products`**
:   List of installed Endpoint Software Blades.

type: keyword


**`checkpoint.cc`**
:   The Carbon Copy address of the email.

type: keyword


**`checkpoint.parent_process_username`**
:   Owner username of the parent process of the process that triggered the attack.

type: keyword


**`checkpoint.process_username`**
:   Owner username of the process that triggered the attack.

type: keyword


**`checkpoint.audit_status`**
:   Audit Status. Can be Success or Failure.

type: keyword


**`checkpoint.objecttable`**
:   Table of affected objects.

type: keyword


**`checkpoint.objecttype`**
:   The type of the affected object.

type: keyword


**`checkpoint.operation_number`**
:   The operation nuber.

type: keyword


**`checkpoint.email_recipients_num`**
:   Amount of recipients whom the mail was sent to.

type: integer


**`checkpoint.suppressed_logs`**
:   Aggregated connections for five minutes on the same source, destination and port.

type: integer


**`checkpoint.blade_name`**
:   Blade name.

type: keyword


**`checkpoint.status`**
:   Ok/Warning/Error.

type: keyword


**`checkpoint.short_desc`**
:   Short description of the process that was executed.

type: keyword


**`checkpoint.long_desc`**
:   More information on the process (usually describing error reason in failure).

type: keyword


**`checkpoint.scan_hosts_hour`**
:   Number of unique hosts during the last hour.

type: integer


**`checkpoint.scan_hosts_day`**
:   Number of unique hosts during the last day.

type: integer


**`checkpoint.scan_hosts_week`**
:   Number of unique hosts during the last week.

type: integer


**`checkpoint.unique_detected_hour`**
:   Detected virus for a specific host during the last hour.

type: integer


**`checkpoint.unique_detected_day`**
:   Detected virus for a specific host during the last day.

type: integer


**`checkpoint.unique_detected_week`**
:   Detected virus for a specific host during the last week.

type: integer


**`checkpoint.scan_mail`**
:   Number of emails that were scanned by "AB malicious activity" engine.

type: integer


**`checkpoint.additional_ip`**
:   DNS host name.

type: keyword


**`checkpoint.description`**
:   Additional explanation how the security gateway enforced the connection.

type: keyword


**`checkpoint.email_spam_category`**
:   Email categories. Possible values: spam/not spam/phishing.

type: keyword


**`checkpoint.email_control_analysis`**
:   Message classification, received from spam vendor engine.

type: keyword


**`checkpoint.scan_results`**
:   "Infected"/description of a failure.

type: keyword


**`checkpoint.original_queue_id`**
:   Original postfix email queue id.

type: keyword


**`checkpoint.risk`**
:   Risk level we got from the engine.

type: keyword


**`checkpoint.roles`**
:   The role of identity.

type: keyword


**`checkpoint.observable_name`**
:   IOC observable signature name.

type: keyword


**`checkpoint.observable_id`**
:   IOC observable signature id.

type: keyword


**`checkpoint.observable_comment`**
:   IOC observable signature description.

type: keyword


**`checkpoint.indicator_name`**
:   IOC indicator name.

type: keyword


**`checkpoint.indicator_description`**
:   IOC indicator description.

type: keyword


**`checkpoint.indicator_reference`**
:   IOC indicator reference.

type: keyword


**`checkpoint.indicator_uuid`**
:   IOC indicator uuid.

type: keyword


**`checkpoint.app_desc`**
:   Application description.

type: keyword


**`checkpoint.app_id`**
:   Application ID.

type: integer


**`checkpoint.app_sig_id`**
:   IOC indicator description.

type: keyword


**`checkpoint.certificate_resource`**
:   HTTPS resource Possible values: SNI or domain name (DN).

type: keyword


**`checkpoint.certificate_validation`**
:   Precise error, describing HTTPS certificate failure under "HTTPS categorize websites" feature.

type: keyword


**`checkpoint.browse_time`**
:   Application session browse time.

type: keyword


**`checkpoint.limit_requested`**
:   Indicates whether data limit was requested for the session.

type: integer


**`checkpoint.limit_applied`**
:   Indicates whether the session was actually date limited.

type: integer


**`checkpoint.dropped_total`**
:   Amount of dropped packets (both incoming and outgoing).

type: integer


**`checkpoint.client_type_os`**
:   Client OS detected in the HTTP request.

type: keyword


**`checkpoint.name`**
:   Application name.

type: keyword


**`checkpoint.properties`**
:   Application categories.

type: keyword


**`checkpoint.sig_id`**
:   Application’s signature ID which how it was detected by.

type: keyword


**`checkpoint.desc`**
:   Override application description.

type: keyword


**`checkpoint.referrer_self_uid`**
:   UUID of the current log.

type: keyword


**`checkpoint.referrer_parent_uid`**
:   Log UUID of the referring application.

type: keyword


**`checkpoint.needs_browse_time`**
:   Browse time required for the connection.

type: integer


**`checkpoint.cluster_info`**
:   Cluster information. Possible options: Failover reason/cluster state changes/CP cluster or 3rd party.

type: keyword


**`checkpoint.sync`**
:   Sync status and the reason (stable, at risk).

type: keyword


**`checkpoint.file_direction`**
:   File direction. Possible options: upload/download.

type: keyword


**`checkpoint.invalid_file_size`**
:   File_size field is valid only if this field is set to 0.

type: integer


**`checkpoint.top_archive_file_name`**
:   In case of archive file: the file that was sent/received.

type: keyword


**`checkpoint.data_type_name`**
:   Data type in rulebase that was matched.

type: keyword


**`checkpoint.specific_data_type_name`**
:   Compound/Group scenario, data type that was matched.

type: keyword


**`checkpoint.word_list`**
:   Words matched by data type.

type: keyword


**`checkpoint.info`**
:   Special log message.

type: keyword


**`checkpoint.outgoing_url`**
:   URL related to this log (for HTTP).

type: keyword


**`checkpoint.dlp_rule_name`**
:   Matched rule name.

type: keyword


**`checkpoint.dlp_recipients`**
:   Mail recipients.

type: keyword


**`checkpoint.dlp_subject`**
:   Mail subject.

type: keyword


**`checkpoint.dlp_word_list`**
:   Phrases matched by data type.

type: keyword


**`checkpoint.dlp_template_score`**
:   Template data type match score.

type: keyword


**`checkpoint.message_size`**
:   Mail/post size.

type: integer


**`checkpoint.dlp_incident_uid`**
:   Unique ID of the matched rule.

type: keyword


**`checkpoint.dlp_related_incident_uid`**
:   Other ID related to this one.

type: keyword


**`checkpoint.dlp_data_type_name`**
:   Matched data type.

type: keyword


**`checkpoint.dlp_data_type_uid`**
:   Unique ID of the matched data type.

type: keyword


**`checkpoint.dlp_violation_description`**
:   Violation descriptions described in the rulebase.

type: keyword


**`checkpoint.dlp_relevant_data_types`**
:   In case of Compound/Group: the inner data types that were matched.

type: keyword


**`checkpoint.dlp_action_reason`**
:   Action chosen reason.

type: keyword


**`checkpoint.dlp_categories`**
:   Data type category.

type: keyword


**`checkpoint.dlp_transint`**
:   HTTP/SMTP/FTP.

type: keyword


**`checkpoint.duplicate`**
:   Log marked as duplicated, when mail is split and the Security Gateway sees it twice.

type: keyword


**`checkpoint.incident_extension`**
:   Matched data type.

type: keyword


**`checkpoint.matched_file`**
:   Unique ID of the matched data type.

type: keyword


**`checkpoint.matched_file_text_segments`**
:   Fingerprint: number of text segments matched by this traffic.

type: integer


**`checkpoint.matched_file_percentage`**
:   Fingerprint: match percentage of the traffic.

type: integer


**`checkpoint.dlp_additional_action`**
:   Watermark/None.

type: keyword


**`checkpoint.dlp_watermark_profile`**
:   Watermark which was applied.

type: keyword


**`checkpoint.dlp_repository_id`**
:   ID of scanned repository.

type: keyword


**`checkpoint.dlp_repository_root_path`**
:   Repository path.

type: keyword


**`checkpoint.scan_id`**
:   Sequential number of scan.

type: keyword


**`checkpoint.special_properties`**
:   If this field is set to *1* the log will not be shown (in use for monitoring scan progress).

type: integer


**`checkpoint.dlp_repository_total_size`**
:   Repository size.

type: integer


**`checkpoint.dlp_repository_files_number`**
:   Number of files in repository.

type: integer


**`checkpoint.dlp_repository_scanned_files_number`**
:   Number of scanned files in repository.

type: integer


**`checkpoint.duration`**
:   Scan duration.

type: keyword


**`checkpoint.dlp_fingerprint_long_status`**
:   Scan status - long format.

type: keyword


**`checkpoint.dlp_fingerprint_short_status`**
:   Scan status - short format.

type: keyword


**`checkpoint.dlp_repository_directories_number`**
:   Number of directories in repository.

type: integer


**`checkpoint.dlp_repository_unreachable_directories_number`**
:   Number of directories the Security Gateway was unable to read.

type: integer


**`checkpoint.dlp_fingerprint_files_number`**
:   Number of successfully scanned files in repository.

type: integer


**`checkpoint.dlp_repository_skipped_files_number`**
:   Skipped number of files because of configuration.

type: integer


**`checkpoint.dlp_repository_scanned_directories_number`**
:   Amount of directories scanned.

type: integer


**`checkpoint.number_of_errors`**
:   Number of files that were not  scanned due to an error.

type: integer


**`checkpoint.next_scheduled_scan_date`**
:   Next scan scheduled time according to time object.

type: keyword


**`checkpoint.dlp_repository_scanned_total_size`**
:   Size scanned.

type: integer


**`checkpoint.dlp_repository_reached_directories_number`**
:   Number of scanned directories in repository.

type: integer


**`checkpoint.dlp_repository_not_scanned_directories_percentage`**
:   Percentage of directories the Security Gateway was unable to read.

type: integer


**`checkpoint.speed`**
:   Current scan speed.

type: integer


**`checkpoint.dlp_repository_scan_progress`**
:   Scan percentage.

type: integer


**`checkpoint.sub_policy_name`**
:   Layer name.

type: keyword


**`checkpoint.sub_policy_uid`**
:   Layer uid.

type: keyword


**`checkpoint.fw_message`**
:   Used for various firewall errors.

type: keyword


**`checkpoint.message`**
:   ISP link has failed.

type: keyword


**`checkpoint.isp_link`**
:   Name of ISP link.

type: keyword


**`checkpoint.fw_subproduct`**
:   Can be vpn/non vpn.

type: keyword


**`checkpoint.sctp_error`**
:   Error information, what caused sctp to fail on out_of_state.

type: keyword


**`checkpoint.chunk_type`**
:   Chunck of the sctp stream.

type: keyword


**`checkpoint.sctp_association_state`**
:   The bad state you were trying to update to.

type: keyword


**`checkpoint.tcp_packet_out_of_state`**
:   State violation.

type: keyword


**`checkpoint.tcp_flags`**
:   TCP packet flags (SYN, ACK, etc.,).

type: keyword


**`checkpoint.connectivity_level`**
:   Log for a new connection in wire mode.

type: keyword


**`checkpoint.ip_option`**
:   IP option that was dropped.

type: integer


**`checkpoint.tcp_state`**
:   Log reinting a tcp state change.

type: keyword


**`checkpoint.expire_time`**
:   Connection closing time.

type: keyword


**`checkpoint.icmp_type`**
:   In case a connection is ICMP, type info will be added to the log.

type: integer


**`checkpoint.icmp_code`**
:   In case a connection is ICMP, code info will be added to the log.

type: integer


**`checkpoint.rpc_prog`**
:   Log for new RPC state - prog values.

type: integer


**`checkpoint.dce-rpc_interface_uuid`**
:   Log for new RPC state - UUID values

type: keyword


**`checkpoint.elapsed`**
:   Time passed since start time.

type: keyword


**`checkpoint.icmp`**
:   Number of packets, received by the client.

type: keyword


**`checkpoint.capture_uuid`**
:   UUID generated for the capture. Used when enabling the capture when logging.

type: keyword


**`checkpoint.diameter_app_ID`**
:   The ID of diameter application.

type: integer


**`checkpoint.diameter_cmd_code`**
:   Diameter not allowed application command id.

type: integer


**`checkpoint.diameter_msg_type`**
:   Diameter message type.

type: keyword


**`checkpoint.cp_message`**
:   Used to log a general message.

type: integer


**`checkpoint.log_delay`**
:   Time left before deleting template.

type: integer


**`checkpoint.attack_status`**
:   In case of a malicious event on an endpoint computer, the status of the attack.

type: keyword


**`checkpoint.impacted_files`**
:   In case of an infection on an endpoint computer, the list of files that the malware impacted.

type: keyword


**`checkpoint.remediated_files`**
:   In case of an infection and a successful cleaning of that infection, this is a list of remediated files on the computer.

type: keyword


**`checkpoint.triggered_by`**
:   The name of the mechanism that triggered the Software Blade to enforce a protection.

type: keyword


**`checkpoint.https_inspection_rule_id`**
:   ID of the matched rule.

type: keyword


**`checkpoint.https_inspection_rule_name`**
:   Name of the matched rule.

type: keyword


**`checkpoint.app_properties`**
:   List of all found categories.

type: keyword


**`checkpoint.https_validation`**
:   Precise error, describing HTTPS inspection failure.

type: keyword


**`checkpoint.https_inspection_action`**
:   HTTPS inspection action (Inspect/Bypass/Error).

type: keyword


**`checkpoint.icap_service_id`**
:   Service ID, can work with multiple servers, treated as services.

type: integer


**`checkpoint.icap_server_name`**
:   Server name.

type: keyword


**`checkpoint.internal_error`**
:   Internal error, for troubleshooting

type: keyword


**`checkpoint.icap_more_info`**
:   Free text for verdict.

type: integer


**`checkpoint.reply_status`**
:   ICAP reply status code, e.g. 200 or 204.

type: integer


**`checkpoint.icap_server_service`**
:   Service name, as given in the ICAP URI

type: keyword


**`checkpoint.mirror_and_decrypt_type`**
:   Information about decrypt and forward. Possible values: Mirror only, Decrypt and mirror, Partial mirroring (HTTPS inspection Bypass).

type: keyword


**`checkpoint.interface_name`**
:   Designated interface for mirror And decrypt.

type: keyword


**`checkpoint.session_uid`**
:   HTTP session-id.

type: keyword


**`checkpoint.broker_publisher`**
:   IP address of the broker publisher who shared the session information.

type: ip


**`checkpoint.src_user_dn`**
:   User distinguished name connected to source IP.

type: keyword


**`checkpoint.proxy_user_name`**
:   User name connected to proxy IP.

type: keyword


**`checkpoint.proxy_machine_name`**
:   Machine name connected to proxy IP.

type: integer


**`checkpoint.proxy_user_dn`**
:   User distinguished name connected to proxy IP.

type: keyword


**`checkpoint.query`**
:   DNS query.

type: keyword


**`checkpoint.dns_query`**
:   DNS query.

type: keyword


**`checkpoint.inspection_item`**
:   Blade element performed inspection.

type: keyword


**`checkpoint.performance_impact`**
:   Protection performance impact.

type: integer


**`checkpoint.inspection_category`**
:   Inspection category: protocol anomaly, signature etc.

type: keyword


**`checkpoint.inspection_profile`**
:   Profile which the activated protection belongs to.

type: keyword


**`checkpoint.summary`**
:   Summary message of a non-compliant DNS traffic drops or detects.

type: keyword


**`checkpoint.question_rdata`**
:   List of question records domains.

type: keyword


**`checkpoint.answer_rdata`**
:   List of answer resource records to the questioned domains.

type: keyword


**`checkpoint.authority_rdata`**
:   List of authoritative servers.

type: keyword


**`checkpoint.additional_rdata`**
:   List of additional resource records.

type: keyword


**`checkpoint.files_names`**
:   List of files requested by FTP.

type: keyword


**`checkpoint.ftp_user`**
:   FTP username.

type: keyword


**`checkpoint.mime_from`**
:   Sender’s address.

type: keyword


**`checkpoint.mime_to`**
:   List of receiver address.

type: keyword


**`checkpoint.bcc`**
:   List of BCC addresses.

type: keyword


**`checkpoint.content_type`**
:   Mail content type. Possible values: application/msword, text/html, image/gif etc.

type: keyword


**`checkpoint.user_agent`**
:   String identifying requesting software user agent.

type: keyword


**`checkpoint.referrer`**
:   Referrer HTTP request header, previous web page address.

type: keyword


**`checkpoint.http_location`**
:   Response header, indicates the URL to redirect a page to.

type: keyword


**`checkpoint.content_disposition`**
:   Indicates how the content is expected to be displayed inline in the browser.

type: keyword


**`checkpoint.via`**
:   Via header is added by proxies for tracking purposes to avoid sending reqests in loop.

type: keyword


**`checkpoint.http_server`**
:   Server HTTP header value, contains information about the software used by the origin server, which handles the request.

type: keyword


**`checkpoint.content_length`**
:   Indicates the size of the entity-body of the HTTP header.

type: keyword


**`checkpoint.authorization`**
:   Authorization HTTP header value.

type: keyword


**`checkpoint.http_host`**
:   Domain name of the server that the HTTP request is sent to.

type: keyword


**`checkpoint.inspection_settings_log`**
:   Indicats that the log was released by inspection settings.

type: keyword


**`checkpoint.cvpn_resource`**
:   Mobile Access application.

type: keyword


**`checkpoint.cvpn_category`**
:   Mobile Access application type.

type: keyword


**`checkpoint.url`**
:   Translated URL.

type: keyword


**`checkpoint.reject_id`**
:   A reject ID that corresponds to the one presented in the Mobile Access error page.

type: keyword


**`checkpoint.fs-proto`**
:   The file share protocol used in mobile acess file share application.

type: keyword


**`checkpoint.app_package`**
:   Unique identifier of the application on the protected mobile device.

type: keyword


**`checkpoint.appi_name`**
:   Name of application downloaded on the protected mobile device.

type: keyword


**`checkpoint.app_repackaged`**
:   Indicates whether the original application was repackage not by the official developer.

type: keyword


**`checkpoint.app_sid_id`**
:   Unique SHA identifier of a mobile application.

type: keyword


**`checkpoint.app_version`**
:   Version of the application downloaded on the protected mobile device.

type: keyword


**`checkpoint.developer_certificate_name`**
:   Name of the developer’s certificate that was used to sign the mobile application.

type: keyword


**`checkpoint.email_control`**
:   Engine name.

type: keyword


**`checkpoint.email_message_id`**
:   Email session id (uniqe ID of the mail).

type: keyword


**`checkpoint.email_queue_id`**
:   Postfix email queue id.

type: keyword


**`checkpoint.email_queue_name`**
:   Postfix email queue name.

type: keyword


**`checkpoint.file_name`**
:   Malicious file name.

type: keyword


**`checkpoint.failure_reason`**
:   MTA failure description.

type: keyword


**`checkpoint.email_headers`**
:   String containing all the email headers.

type: keyword


**`checkpoint.arrival_time`**
:   Email arrival timestamp.

type: keyword


**`checkpoint.email_status`**
:   Describes the email’s state. Possible options: delivered, deferred, skipped, bounced, hold, new, scan_started, scan_ended

type: keyword


**`checkpoint.status_update`**
:   Last time log was updated.

type: keyword


**`checkpoint.delivery_time`**
:   Timestamp of when email was delivered (MTA finished handling the email.

type: keyword


**`checkpoint.links_num`**
:   Number of links in the mail.

type: integer


**`checkpoint.attachments_num`**
:   Number of attachments in the mail.

type: integer


**`checkpoint.email_content`**
:   Mail contents. Possible options: attachments/links & attachments/links/text only.

type: keyword


**`checkpoint.allocated_ports`**
:   Amount of allocated ports.

type: integer


**`checkpoint.capacity`**
:   Capacity of the ports.

type: integer


**`checkpoint.ports_usage`**
:   Percentage of allocated ports.

type: integer


**`checkpoint.nat_exhausted_pool`**
:   4-tuple of an exhausted pool.

type: keyword


**`checkpoint.nat_rulenum`**
:   NAT rulebase first matched rule.

type: integer


**`checkpoint.nat_addtnl_rulenum`**
:   When matching 2 automatic rules , second rule match will be shown otherwise field will be 0.

type: integer


**`checkpoint.message_info`**
:   Used for information messages, for example:NAT connection has ended.

type: keyword


**`checkpoint.nat46`**
:   NAT 46 status, in most cases "enabled".

type: keyword


**`checkpoint.end_time`**
:   TCP connection end time.

type: keyword


**`checkpoint.tcp_end_reason`**
:   Reason for TCP connection closure.

type: keyword


**`checkpoint.cgnet`**
:   Describes NAT allocation for specific subscriber.

type: keyword


**`checkpoint.subscriber`**
:   Source IP before CGNAT.

type: ip


**`checkpoint.hide_ip`**
:   Source IP which will be used after CGNAT.

type: ip


**`checkpoint.int_start`**
:   Subscriber start int which will be used for NAT.

type: integer


**`checkpoint.int_end`**
:   Subscriber end int which will be used for NAT.

type: integer


**`checkpoint.packet_amount`**
:   Amount of packets dropped.

type: integer


**`checkpoint.monitor_reason`**
:   Aggregated logs of monitored packets.

type: keyword


**`checkpoint.drops_amount`**
:   Amount of multicast packets dropped.

type: integer


**`checkpoint.securexl_message`**
:   Two options for a SecureXL message: 1. Missed accounting records after heavy load on logging system. 2. FW log message regarding a packet drop.

type: keyword


**`checkpoint.conns_amount`**
:   Connections amount of aggregated log info.

type: integer


**`checkpoint.scope`**
:   IP related to the attack.

type: keyword


**`checkpoint.analyzed_on`**
:   Check Point ThreatCloud / emulator name.

type: keyword


**`checkpoint.detected_on`**
:   System and applications version the file was emulated on.

type: keyword


**`checkpoint.dropped_file_name`**
:   List of names dropped from the original file.

type: keyword


**`checkpoint.dropped_file_type`**
:   List of file types dropped from the original file.

type: keyword


**`checkpoint.dropped_file_hash`**
:   List of file hashes dropped from the original file.

type: keyword


**`checkpoint.dropped_file_verdict`**
:   List of file verdics dropped from the original file.

type: keyword


**`checkpoint.emulated_on`**
:   Images the files were emulated on.

type: keyword


**`checkpoint.extracted_file_type`**
:   Types of extracted files in case of an archive.

type: keyword


**`checkpoint.extracted_file_names`**
:   Names of extracted files in case of an archive.

type: keyword


**`checkpoint.extracted_file_hash`**
:   Archive hash in case of extracted files.

type: keyword


**`checkpoint.extracted_file_verdict`**
:   Verdict of extracted files in case of an archive.

type: keyword


**`checkpoint.extracted_file_uid`**
:   UID of extracted files in case of an archive.

type: keyword


**`checkpoint.mitre_initial_access`**
:   The adversary is trying to break into your network.

type: keyword


**`checkpoint.mitre_execution`**
:   The adversary is trying to run malicious code.

type: keyword


**`checkpoint.mitre_persistence`**
:   The adversary is trying to maintain his foothold.

type: keyword


**`checkpoint.mitre_privilege_escalation`**
:   The adversary is trying to gain higher-level permissions.

type: keyword


**`checkpoint.mitre_defense_evasion`**
:   The adversary is trying to avoid being detected.

type: keyword


**`checkpoint.mitre_credential_access`**
:   The adversary is trying to steal account names and passwords.

type: keyword


**`checkpoint.mitre_discovery`**
:   The adversary is trying to expose information about your environment.

type: keyword


**`checkpoint.mitre_lateral_movement`**
:   The adversary is trying to explore your environment.

type: keyword


**`checkpoint.mitre_collection`**
:   The adversary is trying to collect data of interest to achieve his goal.

type: keyword


**`checkpoint.mitre_command_and_control`**
:   The adversary is trying to communicate with compromised systems in order to control them.

type: keyword


**`checkpoint.mitre_exfiltration`**
:   The adversary is trying to steal data.

type: keyword


**`checkpoint.mitre_impact`**
:   The adversary is trying to manipulate, interrupt, or destroy your systems and data.

type: keyword


**`checkpoint.parent_file_hash`**
:   Archive’s hash in case of extracted files.

type: keyword


**`checkpoint.parent_file_name`**
:   Archive’s name in case of extracted files.

type: keyword


**`checkpoint.parent_file_uid`**
:   Archive’s UID in case of extracted files.

type: keyword


**`checkpoint.similiar_iocs`**
:   Other IoCs similar to the ones found, related to the malicious file.

type: keyword


**`checkpoint.similar_hashes`**
:   Hashes found similar to the malicious file.

type: keyword


**`checkpoint.similar_strings`**
:   Strings found similar to the malicious file.

type: keyword


**`checkpoint.similar_communication`**
:   Network action found similar to the malicious file.

type: keyword


**`checkpoint.te_verdict_determined_by`**
:   Emulators determined file verdict.

type: keyword


**`checkpoint.packet_capture_unique_id`**
:   Identifier of the packet capture files.

type: keyword


**`checkpoint.total_attachments`**
:   The number of attachments in an email.

type: integer


**`checkpoint.additional_info`**
:   ID of original file/mail which are sent by admin.

type: keyword


**`checkpoint.content_risk`**
:   File risk.

type: integer


**`checkpoint.operation`**
:   Operation made by Threat Extraction.

type: keyword


**`checkpoint.scrubbed_content`**
:   Active content that was found.

type: keyword


**`checkpoint.scrub_time`**
:   Extraction process duration.

type: keyword


**`checkpoint.scrub_download_time`**
:   File download time from resource.

type: keyword


**`checkpoint.scrub_total_time`**
:   Threat extraction total file handling time.

type: keyword


**`checkpoint.scrub_activity`**
:   The result of the extraction

type: keyword


**`checkpoint.watermark`**
:   Reports whether watermark is added to the cleaned file.

type: keyword


**`checkpoint.snid`**
:   The Check Point session ID.

type: keyword


**`checkpoint.source_object`**
:   Matched object name on source column.

type: keyword


**`checkpoint.destination_object`**
:   Matched object name on destination column.

type: keyword


**`checkpoint.drop_reason`**
:   Drop reason description.

type: keyword


**`checkpoint.hit`**
:   Number of hits on a rule.

type: integer


**`checkpoint.rulebase_id`**
:   Layer number.

type: integer


**`checkpoint.first_hit_time`**
:   First hit time in current interval.

type: integer


**`checkpoint.last_hit_time`**
:   Last hit time in current interval.

type: integer


**`checkpoint.rematch_info`**
:   Information sent when old connections cannot be matched during policy installation.

type: keyword


**`checkpoint.last_rematch_time`**
:   Connection rematched time.

type: keyword


**`checkpoint.action_reason`**
:   Connection drop reason.

type: integer


**`checkpoint.action_reason_msg`**
:   Connection drop reason message.

type: keyword


**`checkpoint.c_bytes`**
:   Boolean value indicates whether bytes sent from the client side are used.

type: integer


**`checkpoint.context_num`**
:   Serial number of the log for a specific connection.

type: integer


**`checkpoint.match_id`**
:   Private key of the rule

type: integer


**`checkpoint.alert`**
:   Alert level of matched rule (for connection logs).

type: keyword


**`checkpoint.parent_rule`**
:   Parent rule number, in case of inline layer.

type: integer


**`checkpoint.match_fk`**
:   Rule number.

type: integer


**`checkpoint.dropped_outgoing`**
:   Number of outgoing bytes dropped when using UP-limit feature.

type: integer


**`checkpoint.dropped_incoming`**
:   Number of incoming bytes dropped when using UP-limit feature.

type: integer


**`checkpoint.media_type`**
:   Media used (audio, video, etc.)

type: keyword


**`checkpoint.sip_reason`**
:   Explains why *source_ip* isn’t allowed to redirect (handover).

type: keyword


**`checkpoint.voip_method`**
:   Registration request.

type: keyword


**`checkpoint.registered_ip-phones`**
:   Registered IP-Phones.

type: keyword


**`checkpoint.voip_reg_user_type`**
:   Registered IP-Phone type.

type: keyword


**`checkpoint.voip_call_id`**
:   Call-ID.

type: keyword


**`checkpoint.voip_reg_int`**
:   Registration port.

type: integer


**`checkpoint.voip_reg_ipp`**
:   Registration IP protocol.

type: integer


**`checkpoint.voip_reg_period`**
:   Registration period.

type: integer


**`checkpoint.voip_log_type`**
:   VoIP log types. Possible values: reject, call, registration.

type: keyword


**`checkpoint.src_phone_number`**
:   Source IP-Phone.

type: keyword


**`checkpoint.voip_from_user_type`**
:   Source IP-Phone type.

type: keyword


**`checkpoint.dst_phone_number`**
:   Destination IP-Phone.

type: keyword


**`checkpoint.voip_to_user_type`**
:   Destination IP-Phone type.

type: keyword


**`checkpoint.voip_call_dir`**
:   Call direction: in/out.

type: keyword


**`checkpoint.voip_call_state`**
:   Call state. Possible values: in/out.

type: keyword


**`checkpoint.voip_call_term_time`**
:   Call termination time stamp.

type: keyword


**`checkpoint.voip_duration`**
:   Call duration (seconds).

type: keyword


**`checkpoint.voip_media_port`**
:   Media int.

type: keyword


**`checkpoint.voip_media_ipp`**
:   Media IP protocol.

type: keyword


**`checkpoint.voip_est_codec`**
:   Estimated codec.

type: keyword


**`checkpoint.voip_exp`**
:   Expiration.

type: integer


**`checkpoint.voip_attach_sz`**
:   Attachment size.

type: integer


**`checkpoint.voip_attach_action_info`**
:   Attachment action Info.

type: keyword


**`checkpoint.voip_media_codec`**
:   Estimated codec.

type: keyword


**`checkpoint.voip_reject_reason`**
:   Reject reason.

type: keyword


**`checkpoint.voip_reason_info`**
:   Information.

type: keyword


**`checkpoint.voip_config`**
:   Configuration.

type: keyword


**`checkpoint.voip_reg_server`**
:   Registrar server IP address.

type: ip


**`checkpoint.scv_user`**
:   Username whose packets are dropped on SCV.

type: keyword


**`checkpoint.scv_message_info`**
:   Drop reason.

type: keyword


**`checkpoint.ppp`**
:   Authentication status.

type: keyword


**`checkpoint.scheme`**
:   Describes the scheme used for the log.

type: keyword


**`checkpoint.auth_method`**
:   Password authentication protocol used (PAP or EAP).

type: keyword


**`checkpoint.auth_status`**
:   The authentication status for an event.

type: keyword


**`checkpoint.machine`**
:   L2TP machine which triggered the log and the log refers to it.

type: keyword


**`checkpoint.vpn_feature_name`**
:   L2TP /IKE / Link Selection.

type: keyword


**`checkpoint.reject_category`**
:   Authentication failure reason.

type: keyword


**`checkpoint.peer_ip_probing_status_update`**
:   IP address response status.

type: keyword


**`checkpoint.peer_ip`**
:   IP address which the client connects to.

type: keyword


**`checkpoint.peer_gateway`**
:   Main IP of the peer Security Gateway.

type: ip


**`checkpoint.link_probing_status_update`**
:   IP address response status.

type: keyword


**`checkpoint.source_interface`**
:   External Interface name for source interface or Null if not found.

type: keyword


**`checkpoint.next_hop_ip`**
:   Next hop IP address.

type: keyword


**`checkpoint.srckeyid`**
:   Initiator Spi ID.

type: keyword


**`checkpoint.dstkeyid`**
:   Responder Spi ID.

type: keyword


**`checkpoint.encryption_failure`**
:   Message indicating why the encryption failed.

type: keyword


**`checkpoint.ike_ids`**
:   All QM ids.

type: keyword


**`checkpoint.community`**
:   Community name for the IPSec key and the use of the IKEv.

type: keyword


**`checkpoint.ike`**
:   IKEMode (PHASE1, PHASE2, etc..).

type: keyword


**`checkpoint.cookieI`**
:   Initiator cookie.

type: keyword


**`checkpoint.cookieR`**
:   Responder cookie.

type: keyword


**`checkpoint.msgid`**
:   Message ID.

type: keyword


**`checkpoint.methods`**
:   IPSEc methods.

type: keyword


**`checkpoint.connection_uid`**
:   Calculation of md5 of the IP and user name as UID.

type: keyword


**`checkpoint.site_name`**
:   Site name.

type: keyword


**`checkpoint.esod_rule_name`**
:   Unknown rule name.

type: keyword


**`checkpoint.esod_rule_action`**
:   Unknown rule action.

type: keyword


**`checkpoint.esod_rule_type`**
:   Unknown rule type.

type: keyword


**`checkpoint.esod_noncompliance_reason`**
:   Non-compliance reason.

type: keyword


**`checkpoint.esod_associated_policies`**
:   Associated policies.

type: keyword


**`checkpoint.spyware_name`**
:   Spyware name.

type: keyword


**`checkpoint.spyware_type`**
:   Spyware type.

type: keyword


**`checkpoint.anti_virus_type`**
:   Anti virus type.

type: keyword


**`checkpoint.end_user_firewall_type`**
:   End user firewall type.

type: keyword


**`checkpoint.esod_scan_status`**
:   Scan failed.

type: keyword


**`checkpoint.esod_access_status`**
:   Access denied.

type: keyword


**`checkpoint.client_type`**
:   Endpoint Connect.

type: keyword


**`checkpoint.precise_error`**
:   HTTP parser error.

type: keyword


**`checkpoint.method`**
:   HTTP method.

type: keyword


**`checkpoint.trusted_domain`**
:   In case of phishing event, the domain, which the attacker was impersonating.

type: keyword


**`checkpoint.comment`**
:   type: keyword


**`checkpoint.conn_direction`**
:   Connection direction

type: keyword


**`checkpoint.db_ver`**
:   Database version

type: keyword


**`checkpoint.update_status`**
:   Status of database update

type: keyword


