---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-fortinet.html
---

# Fortinet fields [exported-fields-fortinet]

fortinet Module


## fortinet [_fortinet]

Fields from fortinet FortiOS

**`fortinet.file.hash.crc32`**
:   CRC32 Hash of file

type: keyword



## firewall [_firewall]

Module for parsing Fortinet syslog.

**`fortinet.firewall.acct_stat`**
:   Accounting state (RADIUS)

type: keyword


**`fortinet.firewall.acktime`**
:   Alarm Acknowledge Time

type: keyword


**`fortinet.firewall.act`**
:   Action

type: keyword


**`fortinet.firewall.action`**
:   Status of the session

type: keyword


**`fortinet.firewall.activity`**
:   HA activity message

type: keyword


**`fortinet.firewall.addr`**
:   IP Address

type: ip


**`fortinet.firewall.addr_type`**
:   Address Type

type: keyword


**`fortinet.firewall.addrgrp`**
:   Address Group

type: keyword


**`fortinet.firewall.adgroup`**
:   AD Group Name

type: keyword


**`fortinet.firewall.admin`**
:   Admin User

type: keyword


**`fortinet.firewall.age`**
:   Time in seconds - time passed since last seen

type: integer


**`fortinet.firewall.agent`**
:   User agent - eg. agent="Mozilla/5.0"

type: keyword


**`fortinet.firewall.alarmid`**
:   Alarm ID

type: integer


**`fortinet.firewall.alert`**
:   Alert

type: keyword


**`fortinet.firewall.analyticscksum`**
:   The checksum of the file submitted for analytics

type: keyword


**`fortinet.firewall.analyticssubmit`**
:   The flag for analytics submission

type: keyword


**`fortinet.firewall.ap`**
:   Access Point

type: keyword


**`fortinet.firewall.app-type`**
:   Address Type

type: keyword


**`fortinet.firewall.appact`**
:   The security action from app control

type: keyword


**`fortinet.firewall.appid`**
:   Application ID

type: integer


**`fortinet.firewall.applist`**
:   Application Control profile

type: keyword


**`fortinet.firewall.apprisk`**
:   Application Risk Level

type: keyword


**`fortinet.firewall.apscan`**
:   The name of the AP, which scanned and detected the rogue AP

type: keyword


**`fortinet.firewall.apsn`**
:   Access Point

type: keyword


**`fortinet.firewall.apstatus`**
:   Access Point status

type: keyword


**`fortinet.firewall.aptype`**
:   Access Point type

type: keyword


**`fortinet.firewall.assigned`**
:   Assigned IP Address

type: ip


**`fortinet.firewall.assignip`**
:   Assigned IP Address

type: ip


**`fortinet.firewall.attachment`**
:   The flag for email attachement

type: keyword


**`fortinet.firewall.attack`**
:   Attack Name

type: keyword


**`fortinet.firewall.attackcontext`**
:   The trigger patterns and the packetdata with base64 encoding

type: keyword


**`fortinet.firewall.attackcontextid`**
:   Attack context id / total

type: keyword


**`fortinet.firewall.attackid`**
:   Attack ID

type: integer


**`fortinet.firewall.auditid`**
:   Audit ID

type: long


**`fortinet.firewall.auditscore`**
:   The Audit Score

type: keyword


**`fortinet.firewall.audittime`**
:   The time of the audit

type: long


**`fortinet.firewall.authgrp`**
:   Authorization Group

type: keyword


**`fortinet.firewall.authid`**
:   Authentication ID

type: keyword


**`fortinet.firewall.authproto`**
:   The protocol that initiated the authentication

type: keyword


**`fortinet.firewall.authserver`**
:   Authentication server

type: keyword


**`fortinet.firewall.bandwidth`**
:   Bandwidth

type: keyword


**`fortinet.firewall.banned_rule`**
:   NAC quarantine Banned Rule Name

type: keyword


**`fortinet.firewall.banned_src`**
:   NAC quarantine Banned Source IP

type: keyword


**`fortinet.firewall.banword`**
:   Banned word

type: keyword


**`fortinet.firewall.botnetdomain`**
:   Botnet Domain Name

type: keyword


**`fortinet.firewall.botnetip`**
:   Botnet IP Address

type: ip


**`fortinet.firewall.bssid`**
:   Service Set ID

type: keyword


**`fortinet.firewall.call_id`**
:   Caller ID

type: keyword


**`fortinet.firewall.carrier_ep`**
:   The FortiOS Carrier end-point identification

type: keyword


**`fortinet.firewall.cat`**
:   DNS category ID

type: integer


**`fortinet.firewall.category`**
:   Authentication category

type: keyword


**`fortinet.firewall.cc`**
:   CC Email Address

type: keyword


**`fortinet.firewall.cdrcontent`**
:   Cdrcontent

type: keyword


**`fortinet.firewall.centralnatid`**
:   Central NAT ID

type: integer


**`fortinet.firewall.cert`**
:   Certificate

type: keyword


**`fortinet.firewall.cert-type`**
:   Certificate type

type: keyword


**`fortinet.firewall.certhash`**
:   Certificate hash

type: keyword


**`fortinet.firewall.cfgattr`**
:   Configuration attribute

type: keyword


**`fortinet.firewall.cfgobj`**
:   Configuration object

type: keyword


**`fortinet.firewall.cfgpath`**
:   Configuration path

type: keyword


**`fortinet.firewall.cfgtid`**
:   Configuration transaction ID

type: keyword


**`fortinet.firewall.cfgtxpower`**
:   Configuration TX power

type: integer


**`fortinet.firewall.channel`**
:   Wireless Channel

type: integer


**`fortinet.firewall.channeltype`**
:   SSH channel type

type: keyword


**`fortinet.firewall.chassisid`**
:   Chassis ID

type: integer


**`fortinet.firewall.checksum`**
:   The checksum of the scanned file

type: keyword


**`fortinet.firewall.chgheaders`**
:   HTTP Headers

type: keyword


**`fortinet.firewall.cldobjid`**
:   Connector object ID

type: keyword


**`fortinet.firewall.client_addr`**
:   Wifi client address

type: keyword


**`fortinet.firewall.cloudaction`**
:   Cloud Action

type: keyword


**`fortinet.firewall.clouduser`**
:   Cloud User

type: keyword


**`fortinet.firewall.column`**
:   VOIP Column

type: integer


**`fortinet.firewall.command`**
:   CLI Command

type: keyword


**`fortinet.firewall.community`**
:   SNMP Community

type: keyword


**`fortinet.firewall.configcountry`**
:   Configuration country

type: keyword


**`fortinet.firewall.connection_type`**
:   FortiClient Connection Type

type: keyword


**`fortinet.firewall.conserve`**
:   Flag for conserve mode

type: keyword


**`fortinet.firewall.constraint`**
:   WAF http protocol restrictions

type: keyword


**`fortinet.firewall.contentdisarmed`**
:   Email scanned content

type: keyword


**`fortinet.firewall.contenttype`**
:   Content Type from HTTP header

type: keyword


**`fortinet.firewall.cookies`**
:   VPN Cookie

type: keyword


**`fortinet.firewall.count`**
:   Counts of action type

type: integer


**`fortinet.firewall.countapp`**
:   Number of App Ctrl logs associated with the session

type: integer


**`fortinet.firewall.countav`**
:   Number of AV logs associated with the session

type: integer


**`fortinet.firewall.countcifs`**
:   Number of CIFS logs associated with the session

type: integer


**`fortinet.firewall.countdlp`**
:   Number of DLP logs associated with the session

type: integer


**`fortinet.firewall.countdns`**
:   Number of DNS logs associated with the session

type: integer


**`fortinet.firewall.countemail`**
:   Number of email logs associated with the session

type: integer


**`fortinet.firewall.countff`**
:   Number of ff logs associated with the session

type: integer


**`fortinet.firewall.countips`**
:   Number of IPS logs associated with the session

type: integer


**`fortinet.firewall.countssh`**
:   Number of SSH logs associated with the session

type: integer


**`fortinet.firewall.countssl`**
:   Number of SSL logs associated with the session

type: integer


**`fortinet.firewall.countwaf`**
:   Number of WAF logs associated with the session

type: integer


**`fortinet.firewall.countweb`**
:   Number of Web filter logs associated with the session

type: integer


**`fortinet.firewall.cpu`**
:   CPU Usage

type: integer


**`fortinet.firewall.craction`**
:   Client Reputation Action

type: integer


**`fortinet.firewall.criticalcount`**
:   Number of critical ratings

type: integer


**`fortinet.firewall.crl`**
:   Client Reputation Level

type: keyword


**`fortinet.firewall.crlevel`**
:   Client Reputation Level

type: keyword


**`fortinet.firewall.crscore`**
:   Some description

type: integer


**`fortinet.firewall.cveid`**
:   CVE ID

type: keyword


**`fortinet.firewall.daemon`**
:   Daemon name

type: keyword


**`fortinet.firewall.datarange`**
:   Data range for reports

type: keyword


**`fortinet.firewall.date`**
:   Date

type: keyword


**`fortinet.firewall.ddnsserver`**
:   DDNS server

type: ip


**`fortinet.firewall.desc`**
:   Description

type: keyword


**`fortinet.firewall.detectionmethod`**
:   Detection method

type: keyword


**`fortinet.firewall.devcategory`**
:   Device category

type: keyword


**`fortinet.firewall.devintfname`**
:   HA device Interface Name

type: keyword


**`fortinet.firewall.devtype`**
:   Device type

type: keyword


**`fortinet.firewall.dhcp_msg`**
:   DHCP Message

type: keyword


**`fortinet.firewall.dintf`**
:   Destination interface

type: keyword


**`fortinet.firewall.disk`**
:   Assosciated disk

type: keyword


**`fortinet.firewall.disklograte`**
:   Disk logging rate

type: long


**`fortinet.firewall.dlpextra`**
:   DLP extra information

type: keyword


**`fortinet.firewall.docsource`**
:   DLP fingerprint document source

type: keyword


**`fortinet.firewall.domainctrlauthstate`**
:   CIFS domain auth state

type: integer


**`fortinet.firewall.domainctrlauthtype`**
:   CIFS domain auth type

type: integer


**`fortinet.firewall.domainctrldomain`**
:   CIFS domain auth domain

type: keyword


**`fortinet.firewall.domainctrlip`**
:   CIFS Domain IP

type: ip


**`fortinet.firewall.domainctrlname`**
:   CIFS Domain name

type: keyword


**`fortinet.firewall.domainctrlprotocoltype`**
:   CIFS Domain connection protocol

type: integer


**`fortinet.firewall.domainctrlusername`**
:   CIFS Domain username

type: keyword


**`fortinet.firewall.domainfilteridx`**
:   Domain filter ID

type: integer


**`fortinet.firewall.domainfilterlist`**
:   Domain filter name

type: keyword


**`fortinet.firewall.ds`**
:   Direction with distribution system

type: keyword


**`fortinet.firewall.dst_int`**
:   Destination interface

type: keyword


**`fortinet.firewall.dstintfrole`**
:   Destination interface role

type: keyword


**`fortinet.firewall.dstcountry`**
:   Destination country

type: keyword


**`fortinet.firewall.dstdevcategory`**
:   Destination device category

type: keyword


**`fortinet.firewall.dstdevtype`**
:   Destination device type

type: keyword


**`fortinet.firewall.dstfamily`**
:   Destination OS family

type: keyword


**`fortinet.firewall.dsthwvendor`**
:   Destination HW vendor

type: keyword


**`fortinet.firewall.dsthwversion`**
:   Destination HW version

type: keyword


**`fortinet.firewall.dstinetsvc`**
:   Destination interface service

type: keyword


**`fortinet.firewall.dstosname`**
:   Destination OS name

type: keyword


**`fortinet.firewall.dstosversion`**
:   Destination OS version

type: keyword


**`fortinet.firewall.dstserver`**
:   Destination server

type: integer


**`fortinet.firewall.dstssid`**
:   Destination SSID

type: keyword


**`fortinet.firewall.dstswversion`**
:   Destination software version

type: keyword


**`fortinet.firewall.dstunauthusersource`**
:   Destination unauthenticated source

type: keyword


**`fortinet.firewall.dstuuid`**
:   UUID of the Destination IP address

type: keyword


**`fortinet.firewall.duid`**
:   DHCP UID

type: keyword


**`fortinet.firewall.eapolcnt`**
:   EAPOL packet count

type: integer


**`fortinet.firewall.eapoltype`**
:   EAPOL packet type

type: keyword


**`fortinet.firewall.encrypt`**
:   Whether the packet is encrypted or not

type: integer


**`fortinet.firewall.encryption`**
:   Encryption method

type: keyword


**`fortinet.firewall.epoch`**
:   Epoch used for locating file

type: integer


**`fortinet.firewall.espauth`**
:   ESP Authentication

type: keyword


**`fortinet.firewall.esptransform`**
:   ESP Transform

type: keyword


**`fortinet.firewall.eventtype`**
:   UTM Event Type

type: keyword


**`fortinet.firewall.exch`**
:   Mail Exchanges from DNS response answer section

type: keyword


**`fortinet.firewall.exchange`**
:   Mail Exchanges from DNS response answer section

type: keyword


**`fortinet.firewall.expectedsignature`**
:   Expected SSL signature

type: keyword


**`fortinet.firewall.expiry`**
:   FortiGuard override expiry timestamp

type: keyword


**`fortinet.firewall.fams_pause`**
:   Fortinet Analysis and Management Service Pause

type: integer


**`fortinet.firewall.fazlograte`**
:   FortiAnalyzer Logging Rate

type: long


**`fortinet.firewall.fctemssn`**
:   FortiClient Endpoint SSN

type: keyword


**`fortinet.firewall.fctuid`**
:   FortiClient UID

type: keyword


**`fortinet.firewall.field`**
:   NTP status field

type: keyword


**`fortinet.firewall.filefilter`**
:   The filter used to identify the affected file

type: keyword


**`fortinet.firewall.filehashsrc`**
:   Filehash source

type: keyword


**`fortinet.firewall.filtercat`**
:   DLP filter category

type: keyword


**`fortinet.firewall.filteridx`**
:   DLP filter ID

type: integer


**`fortinet.firewall.filtername`**
:   DLP rule name

type: keyword


**`fortinet.firewall.filtertype`**
:   DLP filter type

type: keyword


**`fortinet.firewall.fortiguardresp`**
:   Antispam ESP value

type: keyword


**`fortinet.firewall.forwardedfor`**
:   Email address forwarded

type: keyword


**`fortinet.firewall.fqdn`**
:   FQDN

type: keyword


**`fortinet.firewall.frametype`**
:   Wireless frametype

type: keyword


**`fortinet.firewall.freediskstorage`**
:   Free disk integer

type: integer


**`fortinet.firewall.from`**
:   From email address

type: keyword


**`fortinet.firewall.from_vcluster`**
:   Source virtual cluster number

type: integer


**`fortinet.firewall.fsaverdict`**
:   FSA verdict

type: keyword


**`fortinet.firewall.fwserver_name`**
:   Web proxy server name

type: keyword


**`fortinet.firewall.gateway`**
:   Gateway ip address for PPPoE status report

type: ip


**`fortinet.firewall.green`**
:   Memory status

type: keyword


**`fortinet.firewall.groupid`**
:   User Group ID

type: integer


**`fortinet.firewall.ha-prio`**
:   HA Priority

type: integer


**`fortinet.firewall.ha_group`**
:   HA Group

type: keyword


**`fortinet.firewall.ha_role`**
:   HA Role

type: keyword


**`fortinet.firewall.handshake`**
:   SSL Handshake

type: keyword


**`fortinet.firewall.hash`**
:   Hash value of downloaded file

type: keyword


**`fortinet.firewall.hbdn_reason`**
:   Heartbeat down reason

type: keyword


**`fortinet.firewall.highcount`**
:   Highcount fabric summary

type: integer


**`fortinet.firewall.host`**
:   Hostname

type: keyword


**`fortinet.firewall.iaid`**
:   DHCPv6 id

type: keyword


**`fortinet.firewall.icmpcode`**
:   Destination Port of the ICMP message

type: keyword


**`fortinet.firewall.icmpid`**
:   Source port of the ICMP message

type: keyword


**`fortinet.firewall.icmptype`**
:   The type of ICMP message

type: keyword


**`fortinet.firewall.identifier`**
:   Network traffic identifier

type: integer


**`fortinet.firewall.in_spi`**
:   IPSEC inbound SPI

type: keyword


**`fortinet.firewall.incidentserialno`**
:   Incident serial number

type: integer


**`fortinet.firewall.infected`**
:   Infected MMS

type: integer


**`fortinet.firewall.infectedfilelevel`**
:   DLP infected file level

type: integer


**`fortinet.firewall.informationsource`**
:   Information source

type: keyword


**`fortinet.firewall.init`**
:   IPSEC init stage

type: keyword


**`fortinet.firewall.initiator`**
:   Original login user name for Fortiguard override

type: keyword


**`fortinet.firewall.interface`**
:   Related interface

type: keyword


**`fortinet.firewall.intf`**
:   Related interface

type: keyword


**`fortinet.firewall.invalidmac`**
:   The MAC address with invalid OUI

type: keyword


**`fortinet.firewall.ip`**
:   Related IP

type: ip


**`fortinet.firewall.iptype`**
:   Related IP type

type: keyword


**`fortinet.firewall.keyword`**
:   Keyword used for search

type: keyword


**`fortinet.firewall.kind`**
:   VOIP kind

type: keyword


**`fortinet.firewall.lanin`**
:   LAN incoming traffic in bytes

type: long


**`fortinet.firewall.lanout`**
:   LAN outbound traffic in bytes

type: long


**`fortinet.firewall.lease`**
:   DHCP lease

type: integer


**`fortinet.firewall.license_limit`**
:   Maximum Number of FortiClients for the License

type: keyword


**`fortinet.firewall.limit`**
:   Virtual Domain Resource Limit

type: integer


**`fortinet.firewall.line`**
:   VOIP line

type: keyword


**`fortinet.firewall.live`**
:   Time in seconds

type: integer


**`fortinet.firewall.local`**
:   Local IP for a PPPD Connection

type: ip


**`fortinet.firewall.log`**
:   Log message

type: keyword


**`fortinet.firewall.login`**
:   SSH login

type: keyword


**`fortinet.firewall.lowcount`**
:   Fabric lowcount

type: integer


**`fortinet.firewall.mac`**
:   DHCP mac address

type: keyword


**`fortinet.firewall.malform_data`**
:   VOIP malformed data

type: integer


**`fortinet.firewall.malform_desc`**
:   VOIP malformed data description

type: keyword


**`fortinet.firewall.manuf`**
:   Manufacturer name

type: keyword


**`fortinet.firewall.masterdstmac`**
:   Master mac address for a host with multiple network interfaces

type: keyword


**`fortinet.firewall.mastersrcmac`**
:   The master MAC address for a host that has multiple network interfaces

type: keyword


**`fortinet.firewall.mediumcount`**
:   Fabric medium count

type: integer


**`fortinet.firewall.mem`**
:   Memory usage system statistics

type: integer


**`fortinet.firewall.meshmode`**
:   Wireless mesh mode

type: keyword


**`fortinet.firewall.message_type`**
:   VOIP message type

type: keyword


**`fortinet.firewall.method`**
:   HTTP method

type: keyword


**`fortinet.firewall.mgmtcnt`**
:   The number of unauthorized client flooding managemet frames

type: integer


**`fortinet.firewall.mode`**
:   IPSEC mode

type: keyword


**`fortinet.firewall.module`**
:   PCI-DSS module

type: keyword


**`fortinet.firewall.monitor-name`**
:   Health Monitor Name

type: keyword


**`fortinet.firewall.monitor-type`**
:   Health Monitor Type

type: keyword


**`fortinet.firewall.mpsk`**
:   Wireless MPSK

type: keyword


**`fortinet.firewall.msgproto`**
:   Message Protocol Number

type: keyword


**`fortinet.firewall.mtu`**
:   Max Transmission Unit Value

type: integer


**`fortinet.firewall.name`**
:   Name

type: keyword


**`fortinet.firewall.nat`**
:   NAT IP Address

type: keyword


**`fortinet.firewall.netid`**
:   Connector NetID

type: keyword


**`fortinet.firewall.new_status`**
:   New status on user change

type: keyword


**`fortinet.firewall.new_value`**
:   New Virtual Domain Name

type: keyword


**`fortinet.firewall.newchannel`**
:   New Channel Number

type: integer


**`fortinet.firewall.newchassisid`**
:   New Chassis ID

type: integer


**`fortinet.firewall.newslot`**
:   New Slot Number

type: integer


**`fortinet.firewall.nextstat`**
:   Time interval in seconds for the next statistics.

type: integer


**`fortinet.firewall.nf_type`**
:   Notification Type

type: keyword


**`fortinet.firewall.noise`**
:   Wifi Noise

type: integer


**`fortinet.firewall.old_status`**
:   Original Status

type: keyword


**`fortinet.firewall.old_value`**
:   Original Virtual Domain name

type: keyword


**`fortinet.firewall.oldchannel`**
:   Original channel

type: integer


**`fortinet.firewall.oldchassisid`**
:   Original Chassis Number

type: integer


**`fortinet.firewall.oldslot`**
:   Original Slot Number

type: integer


**`fortinet.firewall.oldsn`**
:   Old Serial number

type: keyword


**`fortinet.firewall.oldwprof`**
:   Old Web Filter Profile

type: keyword


**`fortinet.firewall.onwire`**
:   A flag to indicate if the AP is onwire or not

type: keyword


**`fortinet.firewall.opercountry`**
:   Operating Country

type: keyword


**`fortinet.firewall.opertxpower`**
:   Operating TX power

type: integer


**`fortinet.firewall.osname`**
:   Operating System name

type: keyword


**`fortinet.firewall.osversion`**
:   Operating System version

type: keyword


**`fortinet.firewall.out_spi`**
:   Out SPI

type: keyword


**`fortinet.firewall.outintf`**
:   Out interface

type: keyword


**`fortinet.firewall.passedcount`**
:   Fabric passed count

type: integer


**`fortinet.firewall.passwd`**
:   Changed user password information

type: keyword


**`fortinet.firewall.path`**
:   Path of looped configuration for security fabric

type: keyword


**`fortinet.firewall.peer`**
:   WAN optimization peer

type: keyword


**`fortinet.firewall.peer_notif`**
:   VPN peer notification

type: keyword


**`fortinet.firewall.phase2_name`**
:   VPN phase2 name

type: keyword


**`fortinet.firewall.phone`**
:   VOIP Phone

type: keyword


**`fortinet.firewall.pid`**
:   Process ID

type: integer


**`fortinet.firewall.policytype`**
:   Policy Type

type: keyword


**`fortinet.firewall.poolname`**
:   IP Pool name

type: keyword


**`fortinet.firewall.port`**
:   Log upload error port

type: integer


**`fortinet.firewall.portbegin`**
:   IP Pool port number to begin

type: integer


**`fortinet.firewall.portend`**
:   IP Pool port number to end

type: integer


**`fortinet.firewall.probeproto`**
:   Link Monitor Probe Protocol

type: keyword


**`fortinet.firewall.process`**
:   URL Filter process

type: keyword


**`fortinet.firewall.processtime`**
:   Process time for reports

type: integer


**`fortinet.firewall.profile`**
:   Profile Name

type: keyword


**`fortinet.firewall.profile_vd`**
:   Virtual Domain Name

type: keyword


**`fortinet.firewall.profilegroup`**
:   Profile Group Name

type: keyword


**`fortinet.firewall.profiletype`**
:   Profile Type

type: keyword


**`fortinet.firewall.qtypeval`**
:   DNS question type value

type: integer


**`fortinet.firewall.quarskip`**
:   Quarantine skip explanation

type: keyword


**`fortinet.firewall.quotaexceeded`**
:   If quota has been exceeded

type: keyword


**`fortinet.firewall.quotamax`**
:   Maximum quota allowed - in seconds if time-based - in bytes if traffic-based

type: long


**`fortinet.firewall.quotatype`**
:   Quota type

type: keyword


**`fortinet.firewall.quotaused`**
:   Quota used - in seconds if time-based - in bytes if trafficbased)

type: long


**`fortinet.firewall.radioband`**
:   Radio band

type: keyword


**`fortinet.firewall.radioid`**
:   Radio ID

type: integer


**`fortinet.firewall.radioidclosest`**
:   Radio ID on the AP closest the rogue AP

type: integer


**`fortinet.firewall.radioiddetected`**
:   Radio ID on the AP which detected the rogue AP

type: integer


**`fortinet.firewall.rate`**
:   Wireless rogue rate value

type: keyword


**`fortinet.firewall.rawdata`**
:   Raw data value

type: keyword


**`fortinet.firewall.rawdataid`**
:   Raw data ID

type: keyword


**`fortinet.firewall.rcvddelta`**
:   Received bytes delta

type: keyword


**`fortinet.firewall.reason`**
:   Alert reason

type: keyword


**`fortinet.firewall.received`**
:   Server key exchange received

type: integer


**`fortinet.firewall.receivedsignature`**
:   Server key exchange received signature

type: keyword


**`fortinet.firewall.red`**
:   Memory information in red

type: keyword


**`fortinet.firewall.referralurl`**
:   Web filter referralurl

type: keyword


**`fortinet.firewall.remote`**
:   Remote PPP IP address

type: ip


**`fortinet.firewall.remotewtptime`**
:   Remote Wifi Radius authentication time

type: keyword


**`fortinet.firewall.reporttype`**
:   Report type

type: keyword


**`fortinet.firewall.reqtype`**
:   Request type

type: keyword


**`fortinet.firewall.request_name`**
:   VOIP request name

type: keyword


**`fortinet.firewall.result`**
:   VPN phase result

type: keyword


**`fortinet.firewall.role`**
:   VPN Phase 2 role

type: keyword


**`fortinet.firewall.rssi`**
:   Received signal strength indicator

type: integer


**`fortinet.firewall.rsso_key`**
:   RADIUS SSO attribute value

type: keyword


**`fortinet.firewall.ruledata`**
:   Rule data

type: keyword


**`fortinet.firewall.ruletype`**
:   Rule type

type: keyword


**`fortinet.firewall.scanned`**
:   Number of Scanned MMSs

type: integer


**`fortinet.firewall.scantime`**
:   Scanned time

type: long


**`fortinet.firewall.scope`**
:   FortiGuard Override Scope

type: keyword


**`fortinet.firewall.security`**
:   Wireless rogue security

type: keyword


**`fortinet.firewall.sensitivity`**
:   Sensitivity for document fingerprint

type: keyword


**`fortinet.firewall.sensor`**
:   NAC Sensor Name

type: keyword


**`fortinet.firewall.sentdelta`**
:   Sent bytes delta

type: keyword


**`fortinet.firewall.seq`**
:   Sequence number

type: keyword


**`fortinet.firewall.serial`**
:   WAN optimisation serial

type: keyword


**`fortinet.firewall.serialno`**
:   Serial number

type: keyword


**`fortinet.firewall.server`**
:   AD server FQDN or IP

type: keyword


**`fortinet.firewall.session_id`**
:   Session ID

type: keyword


**`fortinet.firewall.sessionid`**
:   WAD Session ID

type: integer


**`fortinet.firewall.setuprate`**
:   Session Setup Rate

type: long


**`fortinet.firewall.severity`**
:   Severity

type: keyword


**`fortinet.firewall.shaperdroprcvdbyte`**
:   Received bytes dropped by shaper

type: integer


**`fortinet.firewall.shaperdropsentbyte`**
:   Sent bytes dropped by shaper

type: integer


**`fortinet.firewall.shaperperipdropbyte`**
:   Dropped bytes per IP by shaper

type: integer


**`fortinet.firewall.shaperperipname`**
:   Traffic shaper name (per IP)

type: keyword


**`fortinet.firewall.shaperrcvdname`**
:   Traffic shaper name for received traffic

type: keyword


**`fortinet.firewall.shapersentname`**
:   Traffic shaper name for sent traffic

type: keyword


**`fortinet.firewall.shapingpolicyid`**
:   Traffic shaper policy ID

type: integer


**`fortinet.firewall.signal`**
:   Wireless rogue API signal

type: integer


**`fortinet.firewall.size`**
:   Email size in bytes

type: long


**`fortinet.firewall.slot`**
:   Slot number

type: integer


**`fortinet.firewall.sn`**
:   Security fabric serial number

type: keyword


**`fortinet.firewall.snclosest`**
:   SN of the AP closest to the rogue AP

type: keyword


**`fortinet.firewall.sndetected`**
:   SN of the AP which detected the rogue AP

type: keyword


**`fortinet.firewall.snmeshparent`**
:   SN of the mesh parent

type: keyword


**`fortinet.firewall.spi`**
:   IPSEC SPI

type: keyword


**`fortinet.firewall.src_int`**
:   Source interface

type: keyword


**`fortinet.firewall.srcintfrole`**
:   Source interface role

type: keyword


**`fortinet.firewall.srccountry`**
:   Source country

type: keyword


**`fortinet.firewall.srcfamily`**
:   Source family

type: keyword


**`fortinet.firewall.srchwvendor`**
:   Source hardware vendor

type: keyword


**`fortinet.firewall.srchwversion`**
:   Source hardware version

type: keyword


**`fortinet.firewall.srcinetsvc`**
:   Source interface service

type: keyword


**`fortinet.firewall.srcname`**
:   Source name

type: keyword


**`fortinet.firewall.srcserver`**
:   Source server

type: integer


**`fortinet.firewall.srcssid`**
:   Source SSID

type: keyword


**`fortinet.firewall.srcswversion`**
:   Source software version

type: keyword


**`fortinet.firewall.srcuuid`**
:   Source UUID

type: keyword


**`fortinet.firewall.sscname`**
:   SSC name

type: keyword


**`fortinet.firewall.ssid`**
:   Base Service Set ID

type: keyword


**`fortinet.firewall.sslaction`**
:   SSL Action

type: keyword


**`fortinet.firewall.ssllocal`**
:   WAD SSL local

type: keyword


**`fortinet.firewall.sslremote`**
:   WAD SSL remote

type: keyword


**`fortinet.firewall.stacount`**
:   Number of stations/clients

type: integer


**`fortinet.firewall.stage`**
:   IPSEC stage

type: keyword


**`fortinet.firewall.stamac`**
:   802.1x station mac

type: keyword


**`fortinet.firewall.state`**
:   Admin login state

type: keyword


**`fortinet.firewall.status`**
:   Status

type: keyword


**`fortinet.firewall.stitch`**
:   Automation stitch triggered

type: keyword


**`fortinet.firewall.subject`**
:   Email subject

type: keyword


**`fortinet.firewall.submodule`**
:   Configuration Sub-Module Name

type: keyword


**`fortinet.firewall.subservice`**
:   AV subservice

type: keyword


**`fortinet.firewall.subtype`**
:   Log subtype

type: keyword


**`fortinet.firewall.suspicious`**
:   Number of Suspicious MMSs

type: integer


**`fortinet.firewall.switchproto`**
:   Protocol change information

type: keyword


**`fortinet.firewall.sync_status`**
:   The sync status with the master

type: keyword


**`fortinet.firewall.sync_type`**
:   The sync type with the master

type: keyword


**`fortinet.firewall.sysuptime`**
:   System uptime

type: keyword


**`fortinet.firewall.tamac`**
:   the MAC address of Transmitter, if none, then Receiver

type: keyword


**`fortinet.firewall.threattype`**
:   WIDS threat type

type: keyword


**`fortinet.firewall.time`**
:   Time of the event

type: keyword


**`fortinet.firewall.to`**
:   Email to field

type: keyword


**`fortinet.firewall.to_vcluster`**
:   destination virtual cluster number

type: integer


**`fortinet.firewall.total`**
:   Total memory

type: integer


**`fortinet.firewall.totalsession`**
:   Total Number of Sessions

type: integer


**`fortinet.firewall.trace_id`**
:   Session clash trace ID

type: keyword


**`fortinet.firewall.trandisp`**
:   NAT translation type

type: keyword


**`fortinet.firewall.transid`**
:   HTTP transaction ID

type: integer


**`fortinet.firewall.translationid`**
:   DNS filter transaltion ID

type: keyword


**`fortinet.firewall.trigger`**
:   Automation stitch trigger

type: keyword


**`fortinet.firewall.trueclntip`**
:   File filter true client IP

type: ip


**`fortinet.firewall.tunnelid`**
:   IPSEC tunnel ID

type: integer


**`fortinet.firewall.tunnelip`**
:   IPSEC tunnel IP

type: ip


**`fortinet.firewall.tunneltype`**
:   IPSEC tunnel type

type: keyword


**`fortinet.firewall.type`**
:   Module type

type: keyword


**`fortinet.firewall.ui`**
:   Admin authentication UI type

type: keyword


**`fortinet.firewall.unauthusersource`**
:   Unauthenticated user source

type: keyword


**`fortinet.firewall.unit`**
:   Power supply unit

type: integer


**`fortinet.firewall.urlfilteridx`**
:   URL filter ID

type: integer


**`fortinet.firewall.urlfilterlist`**
:   URL filter list

type: keyword


**`fortinet.firewall.urlsource`**
:   URL filter source

type: keyword


**`fortinet.firewall.urltype`**
:   URL filter type

type: keyword


**`fortinet.firewall.used`**
:   Number of Used IPs

type: integer


**`fortinet.firewall.used_for_type`**
:   Connection for the type

type: integer


**`fortinet.firewall.utmaction`**
:   Security action performed by UTM

type: keyword


**`fortinet.firewall.utmref`**
:   Reference to UTM

type: keyword


**`fortinet.firewall.vap`**
:   Virtual AP

type: keyword


**`fortinet.firewall.vapmode`**
:   Virtual AP mode

type: keyword


**`fortinet.firewall.vcluster`**
:   virtual cluster id

type: integer


**`fortinet.firewall.vcluster_member`**
:   Virtual cluster member

type: integer


**`fortinet.firewall.vcluster_state`**
:   Virtual cluster state

type: keyword


**`fortinet.firewall.vd`**
:   Virtual Domain Name

type: keyword


**`fortinet.firewall.vdname`**
:   Virtual Domain Name

type: keyword


**`fortinet.firewall.vendorurl`**
:   Vulnerability scan vendor name

type: keyword


**`fortinet.firewall.version`**
:   Version

type: keyword


**`fortinet.firewall.vip`**
:   Virtual IP

type: keyword


**`fortinet.firewall.virus`**
:   Virus name

type: keyword


**`fortinet.firewall.virusid`**
:   Virus ID (unique virus identifier)

type: integer


**`fortinet.firewall.voip_proto`**
:   VOIP protocol

type: keyword


**`fortinet.firewall.vpn`**
:   VPN description

type: keyword


**`fortinet.firewall.vpntunnel`**
:   IPsec Vpn Tunnel Name

type: keyword


**`fortinet.firewall.vpntype`**
:   The type of the VPN tunnel

type: keyword


**`fortinet.firewall.vrf`**
:   VRF number

type: integer


**`fortinet.firewall.vulncat`**
:   Vulnerability Category

type: keyword


**`fortinet.firewall.vulnid`**
:   Vulnerability ID

type: integer


**`fortinet.firewall.vulnname`**
:   Vulnerability name

type: keyword


**`fortinet.firewall.vwlid`**
:   VWL ID

type: integer


**`fortinet.firewall.vwlquality`**
:   VWL quality

type: keyword


**`fortinet.firewall.vwlservice`**
:   VWL service

type: keyword


**`fortinet.firewall.vwpvlanid`**
:   VWP VLAN ID

type: integer


**`fortinet.firewall.wanin`**
:   WAN incoming traffic in bytes

type: long


**`fortinet.firewall.wanoptapptype`**
:   WAN Optimization Application type

type: keyword


**`fortinet.firewall.wanout`**
:   WAN outgoing traffic in bytes

type: long


**`fortinet.firewall.weakwepiv`**
:   Weak Wep Initiation Vector

type: keyword


**`fortinet.firewall.xauthgroup`**
:   XAuth Group Name

type: keyword


**`fortinet.firewall.xauthuser`**
:   XAuth User Name

type: keyword


**`fortinet.firewall.xid`**
:   Wireless X ID

type: integer


