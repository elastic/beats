---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-sophos.html
---

# sophos fields [exported-fields-sophos]

sophos Module


## sophos.xg [_sophos_xg]

Module for parsing sophosxg syslog.

**`sophos.xg.action`**
:   Event Action

type: keyword


**`sophos.xg.activityname`**
:   Web policy activity that matched and caused the policy result.

type: keyword


**`sophos.xg.ap`**
:   Access Point Serial ID or LocalWifi0 or LocalWifi1.

type: keyword


**`sophos.xg.app_category`**
:   Name of the category under which application falls

type: keyword


**`sophos.xg.app_filter_policy_id`**
:   Application filter policy ID applied on the traffic

type: keyword


**`sophos.xg.app_is_cloud`**
:   Application is Cloud

type: keyword


**`sophos.xg.app_name`**
:   Application name

type: keyword


**`sophos.xg.app_resolved_by`**
:   Application is resolved by signature or synchronized application

type: keyword


**`sophos.xg.app_risk`**
:   Risk level assigned to the application

type: keyword


**`sophos.xg.app_technology`**
:   Technology of the application

type: keyword


**`sophos.xg.appfilter_policy_id`**
:   Application Filter policy applied on the traffic

type: integer


**`sophos.xg.application`**
:   Application name

type: keyword


**`sophos.xg.application_category`**
:   Application is resolved by signature or synchronized application

type: keyword


**`sophos.xg.application_filter_policy`**
:   Application Filter policy applied on the traffic

type: integer


**`sophos.xg.application_name`**
:   Application name

type: keyword


**`sophos.xg.application_risk`**
:   Risk level assigned to the application

type: keyword


**`sophos.xg.application_technology`**
:   Technology of the application

type: keyword


**`sophos.xg.appresolvedby`**
:   Technology of the application

type: keyword


**`sophos.xg.auth_client`**
:   Auth Client

type: keyword


**`sophos.xg.auth_mechanism`**
:   Auth mechanism

type: keyword


**`sophos.xg.av_policy_name`**
:   Malware scanning policy name which is applied on the traffic

type: keyword


**`sophos.xg.backup_mode`**
:   Backup mode

type: keyword


**`sophos.xg.branch_name`**
:   Branch Name

type: keyword


**`sophos.xg.category`**
:   IPS signature category.

type: keyword


**`sophos.xg.category_type`**
:   Type of category under which website falls

type: keyword


**`sophos.xg.classification`**
:   Signature classification

type: keyword


**`sophos.xg.client_host_name`**
:   Client host name

type: keyword


**`sophos.xg.client_physical_address`**
:   Client physical address

type: keyword


**`sophos.xg.clients_conn_ssid`**
:   Number of client connected to the SSID.

type: long


**`sophos.xg.collisions`**
:   collisions

type: long


**`sophos.xg.con_event`**
:   Event Start/Stop

type: keyword


**`sophos.xg.con_id`**
:   Unique identifier of connection

type: integer


**`sophos.xg.configuration`**
:   Configuration

type: float


**`sophos.xg.conn_id`**
:   Unique identifier of connection

type: integer


**`sophos.xg.connectionname`**
:   Connectionname

type: keyword


**`sophos.xg.connectiontype`**
:   Connectiontype

type: keyword


**`sophos.xg.connevent`**
:   Event on which this log is generated

type: keyword


**`sophos.xg.connid`**
:   Connection ID

type: keyword


**`sophos.xg.content_type`**
:   Type of the content

type: keyword


**`sophos.xg.contenttype`**
:   Type of the content

type: keyword


**`sophos.xg.context_match`**
:   Context Match

type: keyword


**`sophos.xg.context_prefix`**
:   Content Prefix

type: keyword


**`sophos.xg.context_suffix`**
:   Context Suffix

type: keyword


**`sophos.xg.cookie`**
:   cookie

type: keyword


**`sophos.xg.date`**
:   Date (yyyy-mm-dd) when the event occurred

type: date


**`sophos.xg.destinationip`**
:   Original destination IP address of traffic

type: ip


**`sophos.xg.device`**
:   device

type: keyword


**`sophos.xg.device_id`**
:   Serial number of the device

type: keyword


**`sophos.xg.device_model`**
:   Model number of the device

type: keyword


**`sophos.xg.device_name`**
:   Model number of the device

type: keyword


**`sophos.xg.dictionary_name`**
:   Dictionary Name

type: keyword


**`sophos.xg.dir_disp`**
:   TPacket direction. Possible values:“org”, “reply”, “”

type: keyword


**`sophos.xg.direction`**
:   Direction

type: keyword


**`sophos.xg.domainname`**
:   Domain from which virus was downloaded

type: keyword


**`sophos.xg.download_file_name`**
:   Download file name

type: keyword


**`sophos.xg.download_file_type`**
:   Download file type

type: keyword


**`sophos.xg.dst_country_code`**
:   Code of the country to which the destination IP belongs

type: keyword


**`sophos.xg.dst_domainname`**
:   Receiver domain name

type: keyword


**`sophos.xg.dst_ip`**
:   Original destination IP address of traffic

type: ip


**`sophos.xg.dst_port`**
:   Original destination port of TCP and UDP traffic

type: integer


**`sophos.xg.dst_zone_type`**
:   Type of destination zone

type: keyword


**`sophos.xg.dstdomain`**
:   Destination Domain

type: keyword


**`sophos.xg.duration`**
:   Durability of traffic (seconds)

type: long


**`sophos.xg.email_subject`**
:   Email Subject

type: keyword


**`sophos.xg.ep_uuid`**
:   Endpoint UUID

type: keyword


**`sophos.xg.ether_type`**
:   ethernet frame type

type: keyword


**`sophos.xg.eventid`**
:   ATP Evenet ID

type: keyword


**`sophos.xg.eventtime`**
:   Event time

type: date


**`sophos.xg.eventtype`**
:   ATP event type

type: keyword


**`sophos.xg.exceptions`**
:   List of the checks excluded by web exceptions.

type: keyword


**`sophos.xg.execution_path`**
:   ATP execution path

type: keyword


**`sophos.xg.extra`**
:   extra

type: keyword


**`sophos.xg.file_name`**
:   Filename

type: keyword


**`sophos.xg.file_path`**
:   File path

type: keyword


**`sophos.xg.file_size`**
:   File Size

type: integer


**`sophos.xg.filename`**
:   File name associated with the event

type: keyword


**`sophos.xg.filepath`**
:   Path of the file containing virus

type: keyword


**`sophos.xg.filesize`**
:   Size of the file that contained virus

type: integer


**`sophos.xg.free`**
:   free

type: integer


**`sophos.xg.from_email_address`**
:   Sender email address

type: keyword


**`sophos.xg.ftp_direction`**
:   Direction of FTP transfer: Upload or Download

type: keyword


**`sophos.xg.ftp_url`**
:   FTP URL from which virus was downloaded

type: keyword


**`sophos.xg.ftpcommand`**
:   FTP command used when virus was found

type: keyword


**`sophos.xg.fw_rule_id`**
:   Firewall Rule ID which is applied on the traffic

type: integer


**`sophos.xg.fw_rule_type`**
:   Firewall rule type which is applied on the traffic

type: keyword


**`sophos.xg.hb_health`**
:   Heartbeat status

type: keyword


**`sophos.xg.hb_status`**
:   Heartbeat status

type: keyword


**`sophos.xg.host`**
:   Host

type: keyword


**`sophos.xg.http_category`**
:   HTTP Category

type: keyword


**`sophos.xg.http_category_type`**
:   HTTP Category Type

type: keyword


**`sophos.xg.httpresponsecode`**
:   code of HTTP response

type: long


**`sophos.xg.iap`**
:   Internet Access policy ID applied on the traffic

type: keyword


**`sophos.xg.icmp_code`**
:   ICMP code of ICMP traffic

type: keyword


**`sophos.xg.icmp_type`**
:   ICMP type of ICMP traffic

type: keyword


**`sophos.xg.idle_cpu`**
:   idle ##

type: float


**`sophos.xg.idp_policy_id`**
:   IPS policy ID which is applied on the traffic

type: integer


**`sophos.xg.idp_policy_name`**
:   IPS policy name i.e. IPS policy name which is applied on the traffic

type: keyword


**`sophos.xg.in_interface`**
:   Interface for incoming traffic, e.g., Port A

type: keyword


**`sophos.xg.interface`**
:   interface

type: keyword


**`sophos.xg.ipaddress`**
:   Ipaddress

type: keyword


**`sophos.xg.ips_policy_id`**
:   IPS policy ID applied on the traffic

type: integer


**`sophos.xg.lease_time`**
:   Lease Time

type: keyword


**`sophos.xg.localgateway`**
:   Localgateway

type: keyword


**`sophos.xg.localnetwork`**
:   Localnetwork

type: keyword


**`sophos.xg.log_component`**
:   Component responsible for logging e.g. Firewall rule

type: keyword


**`sophos.xg.log_id`**
:   Unique 12 characters code (0101011)

type: keyword


**`sophos.xg.log_subtype`**
:   Sub type of event

type: keyword


**`sophos.xg.log_type`**
:   Type of event e.g. firewall event

type: keyword


**`sophos.xg.log_version`**
:   Log Version

type: keyword


**`sophos.xg.login_user`**
:   ATP login user

type: keyword


**`sophos.xg.mailid`**
:   mailid

type: keyword


**`sophos.xg.mailsize`**
:   mailsize

type: integer


**`sophos.xg.message`**
:   Message

type: keyword


**`sophos.xg.mode`**
:   Mode

type: keyword


**`sophos.xg.nat_rule_id`**
:   NAT Rule ID

type: keyword


**`sophos.xg.newversion`**
:   Newversion

type: keyword


**`sophos.xg.oldversion`**
:   Oldversion

type: keyword


**`sophos.xg.out_interface`**
:   Interface for outgoing traffic, e.g., Port B

type: keyword


**`sophos.xg.override_authorizer`**
:   Override authorizer

type: keyword


**`sophos.xg.override_name`**
:   Override name

type: keyword


**`sophos.xg.override_token`**
:   Override token

type: keyword


**`sophos.xg.phpsessid`**
:   PHP session ID

type: keyword


**`sophos.xg.platform`**
:   Platform of the traffic.

type: keyword


**`sophos.xg.policy_type`**
:   Policy type applied to the traffic

type: keyword


**`sophos.xg.priority`**
:   Severity level of traffic

type: keyword


**`sophos.xg.protocol`**
:   Protocol number of traffic

type: keyword


**`sophos.xg.qualifier`**
:   Qualifier

type: keyword


**`sophos.xg.quarantine`**
:   Path and filename of the file quarantined

type: keyword


**`sophos.xg.quarantine_reason`**
:   Quarantine reason

type: keyword


**`sophos.xg.querystring`**
:   querystring

type: keyword


**`sophos.xg.raw_data`**
:   Raw data

type: keyword


**`sophos.xg.received_pkts`**
:   Total number of packets received

type: long


**`sophos.xg.receiveddrops`**
:   received drops

type: long


**`sophos.xg.receivederrors`**
:   received errors

type: keyword


**`sophos.xg.receivedkbits`**
:   received kbits

type: long


**`sophos.xg.recv_bytes`**
:   Total number of bytes received

type: long


**`sophos.xg.red_id`**
:   RED ID

type: keyword


**`sophos.xg.referer`**
:   Referer

type: keyword


**`sophos.xg.remote_ip`**
:   Remote IP

type: ip


**`sophos.xg.remotenetwork`**
:   remotenetwork

type: keyword


**`sophos.xg.reported_host`**
:   Reported Host

type: keyword


**`sophos.xg.reported_ip`**
:   Reported IP

type: keyword


**`sophos.xg.reports`**
:   Reports

type: float


**`sophos.xg.rule_priority`**
:   Priority of IPS policy

type: keyword


**`sophos.xg.sent_bytes`**
:   Total number of bytes sent

type: long


**`sophos.xg.sent_pkts`**
:   Total number of packets sent

type: long


**`sophos.xg.server`**
:   Server

type: keyword


**`sophos.xg.sessionid`**
:   Sessionid

type: keyword


**`sophos.xg.sha1sum`**
:   SHA1 checksum of the item being analyzed

type: keyword


**`sophos.xg.signature`**
:   Signature

type: float


**`sophos.xg.signature_id`**
:   Signature ID

type: keyword


**`sophos.xg.signature_msg`**
:   Signature messsage

type: keyword


**`sophos.xg.site_category`**
:   Site Category

type: keyword


**`sophos.xg.source`**
:   Source

type: keyword


**`sophos.xg.sourceip`**
:   Original source IP address of traffic

type: ip


**`sophos.xg.spamaction`**
:   Spam Action

type: keyword


**`sophos.xg.sqli`**
:   related SQLI caught by the WAF

type: keyword


**`sophos.xg.src_country_code`**
:   Code of the country to which the source IP belongs

type: keyword


**`sophos.xg.src_domainname`**
:   Sender domain name

type: keyword


**`sophos.xg.src_ip`**
:   Original source IP address of traffic

type: ip


**`sophos.xg.src_mac`**
:   Original source MAC address of traffic

type: keyword


**`sophos.xg.src_port`**
:   Original source port of TCP and UDP traffic

type: integer


**`sophos.xg.src_zone_type`**
:   Type of source zone

type: keyword


**`sophos.xg.ssid`**
:   Configured SSID name.

type: keyword


**`sophos.xg.start_time`**
:   Start time

type: date


**`sophos.xg.starttime`**
:   Starttime

type: date


**`sophos.xg.status`**
:   Ultimate status of traffic – Allowed or Denied

type: keyword


**`sophos.xg.status_code`**
:   Status code

type: keyword


**`sophos.xg.subject`**
:   Email subject

type: keyword


**`sophos.xg.syslog_server_name`**
:   Syslog server name.

type: keyword


**`sophos.xg.system_cpu`**
:   system

type: float


**`sophos.xg.target`**
:   Platform of the traffic.

type: keyword


**`sophos.xg.temp`**
:   Temp

type: float


**`sophos.xg.threatname`**
:   ATP threatname

type: keyword


**`sophos.xg.timestamp`**
:   timestamp

type: date


**`sophos.xg.timezone`**
:   Time (hh:mm:ss) when the event occurred

type: keyword


**`sophos.xg.to_email_address`**
:   Receipeint email address

type: keyword


**`sophos.xg.total_memory`**
:   Total Memory

type: integer


**`sophos.xg.trans_dst_ip`**
:   Translated destination IP address for outgoing traffic

type: ip


**`sophos.xg.trans_dst_port`**
:   Translated destination port for outgoing traffic

type: integer


**`sophos.xg.trans_src_ip`**
:   Translated source IP address for outgoing traffic

type: ip


**`sophos.xg.trans_src_port`**
:   Translated source port for outgoing traffic

type: integer


**`sophos.xg.transaction_id`**
:   Transaction ID

type: keyword


**`sophos.xg.transactionid`**
:   Transaction ID of the AV scan.

type: keyword


**`sophos.xg.transmitteddrops`**
:   transmitted drops

type: long


**`sophos.xg.transmittederrors`**
:   transmitted errors

type: keyword


**`sophos.xg.transmittedkbits`**
:   transmitted kbits

type: long


**`sophos.xg.unit`**
:   unit

type: keyword


**`sophos.xg.updatedip`**
:   updatedip

type: ip


**`sophos.xg.upload_file_name`**
:   Upload file name

type: keyword


**`sophos.xg.upload_file_type`**
:   Upload file type

type: keyword


**`sophos.xg.url`**
:   URL from which virus was downloaded

type: keyword


**`sophos.xg.used`**
:   used

type: integer


**`sophos.xg.used_quota`**
:   Used Quota

type: keyword


**`sophos.xg.user`**
:   User

type: keyword


**`sophos.xg.user_cpu`**
:   system

type: float


**`sophos.xg.user_gp`**
:   Group name to which the user belongs.

type: keyword


**`sophos.xg.user_group`**
:   Group name to which the user belongs

type: keyword


**`sophos.xg.user_name`**
:   user_name

type: keyword


**`sophos.xg.users`**
:   Number of users from System Health / Live User events.

type: long


**`sophos.xg.vconn_id`**
:   Connection ID of the master connection

type: integer


**`sophos.xg.virus`**
:   virus name

type: keyword


**`sophos.xg.web_policy_id`**
:   Web policy ID

type: keyword


**`sophos.xg.website`**
:   Website

type: keyword


**`sophos.xg.xss`**
:   related XSS caught by the WAF

type: keyword


