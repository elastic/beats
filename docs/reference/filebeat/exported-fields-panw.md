---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-panw.html
---

# panw fields [exported-fields-panw]

Module for Palo Alto Networks (PAN-OS)


## panw [_panw]

Fields from the panw module.


## panos [_panos]

Fields for the Palo Alto Networks PAN-OS logs.

**`panw.panos.ruleset`**
:   Name of the rule that matched this session.

type: keyword



## source [_source_3]

Fields to extend the top-level source object.

**`panw.panos.source.zone`**
:   Source zone for this session.

type: keyword


**`panw.panos.source.interface`**
:   Source interface for this session.

type: keyword



## nat [_nat]

Post-NAT source address, if source NAT is performed.

**`panw.panos.source.nat.ip`**
:   Post-NAT source IP.

type: ip


**`panw.panos.source.nat.port`**
:   Post-NAT source port.

type: long



## destination [_destination_3]

Fields to extend the top-level destination object.

**`panw.panos.destination.zone`**
:   Destination zone for this session.

type: keyword


**`panw.panos.destination.interface`**
:   Destination interface for this session.

type: keyword



## nat [_nat_2]

Post-NAT destination address, if destination NAT is performed.

**`panw.panos.destination.nat.ip`**
:   Post-NAT destination IP.

type: ip


**`panw.panos.destination.nat.port`**
:   Post-NAT destination port.

type: long


**`panw.panos.endreason`**
:   The reason a session terminated.

type: keyword



## network [_network_2]

Fields to extend the top-level network object.

**`panw.panos.network.pcap_id`**
:   Packet capture ID for a threat.

type: keyword


**`panw.panos.network.nat.community_id`**
:   Community ID flow-hash for the NAT 5-tuple.

type: keyword



## file [_file_3]

Fields to extend the top-level file object.

**`panw.panos.file.hash`**
:   Binary hash for a threat file sent to be analyzed by the WildFire service.

type: keyword



## url [_url_4]

Fields to extend the top-level url object.

**`panw.panos.url.category`**
:   For threat URLs, it’s the URL category. For WildFire, the verdict on the file and is either *malicious*, *grayware*, or *benign*.

type: keyword


**`panw.panos.flow_id`**
:   Internal numeric identifier for each session.

type: keyword


**`panw.panos.sequence_number`**
:   Log entry identifier that is incremented sequentially. Unique for each log type.

type: long


**`panw.panos.threat.resource`**
:   URL or file name for a threat.

type: keyword


**`panw.panos.threat.id`**
:   Palo Alto Networks identifier for the threat.

type: keyword


**`panw.panos.threat.name`**
:   Palo Alto Networks name for the threat.

type: keyword


**`panw.panos.action`**
:   Action taken for the session.

type: keyword


**`panw.panos.type`**
:   Specifies the type of the log


**`panw.panos.sub_type`**
:   Specifies the sub type of the log


**`panw.panos.virtual_sys`**
:   Virtual system instance

type: keyword


**`panw.panos.client_os_ver`**
:   The client device’s OS version.

type: keyword


**`panw.panos.client_os`**
:   The client device’s OS version.

type: keyword


**`panw.panos.client_ver`**
:   The client’s GlobalProtect app version.

type: keyword


**`panw.panos.stage`**
:   A string showing the stage of the connection

type: keyword

example: before-login


**`panw.panos.actionflags`**
:   A bit field indicating if the log was forwarded to Panorama.

type: keyword


**`panw.panos.error`**
:   A string showing that error that has occurred in any event.

type: keyword


**`panw.panos.error_code`**
:   An integer associated with any errors that occurred.

type: integer


**`panw.panos.repeatcnt`**
:   The number of sessions with the same source IP address, destination IP address, application, and subtype that GlobalProtect has detected within the last five seconds.An integer associated with any errors that occurred.

type: integer


**`panw.panos.serial_number`**
:   The serial number of the user’s machine or device.

type: keyword


**`panw.panos.auth_method`**
:   A string showing the authentication type

type: keyword

example: LDAP


**`panw.panos.datasource`**
:   Source from which mapping information is collected.

type: keyword


**`panw.panos.datasourcetype`**
:   Mechanism used to identify the IP/User mappings within a data source.

type: keyword


**`panw.panos.datasourcename`**
:   User-ID source that sends the IP (Port)-User Mapping.

type: keyword


**`panw.panos.factorno`**
:   Indicates the use of primary authentication (1) or additional factors (2, 3).

type: integer


**`panw.panos.factortype`**
:   Vendor used to authenticate a user when Multi Factor authentication is present.

type: keyword


**`panw.panos.factorcompletiontime`**
:   Time the authentication was completed.

type: date


**`panw.panos.ugflags`**
:   Displays whether the user group that was found during user group mapping. Supported values are: User Group Found—Indicates whether the user could be mapped to a group. Duplicate User—Indicates whether duplicate users were found in a user group. Displays N/A if no user group is found.

type: keyword



## device_group_hierarchy [_device_group_hierarchy]

A sequence of identification numbers that indicate the device group’s location within a device group hierarchy. The firewall (or virtual system) generating the log includes the identification number of each ancestor in its device group hierarchy. The shared device group (level 0) is not included in this structure. If the log values are 12, 34, 45, 0, it means that the log was generated by a firewall (or virtual system) that belongs to device group 45, and its ancestors are 34, and 12.

**`panw.panos.device_group_hierarchy.level_1`**
:   A sequence of identification numbers that indicate the device group’s location within a device group hierarchy. The firewall (or virtual system) generating the log includes the identification number of each ancestor in its device group hierarchy. The shared device group (level 0) is not included in this structure. If the log values are 12, 34, 45, 0, it means that the log was generated by a firewall (or virtual system) that belongs to device group 45, and its ancestors are 34, and 12.

type: keyword


**`panw.panos.device_group_hierarchy.level_2`**
:   A sequence of identification numbers that indicate the device group’s location within a device group hierarchy. The firewall (or virtual system) generating the log includes the identification number of each ancestor in its device group hierarchy. The shared device group (level 0) is not included in this structure. If the log values are 12, 34, 45, 0, it means that the log was generated by a firewall (or virtual system) that belongs to device group 45, and its ancestors are 34, and 12.

type: keyword


**`panw.panos.device_group_hierarchy.level_3`**
:   A sequence of identification numbers that indicate the device group’s location within a device group hierarchy. The firewall (or virtual system) generating the log includes the identification number of each ancestor in its device group hierarchy. The shared device group (level 0) is not included in this structure. If the log values are 12, 34, 45, 0, it means that the log was generated by a firewall (or virtual system) that belongs to device group 45, and its ancestors are 34, and 12.

type: keyword


**`panw.panos.device_group_hierarchy.level_4`**
:   A sequence of identification numbers that indicate the device group’s location within a device group hierarchy. The firewall (or virtual system) generating the log includes the identification number of each ancestor in its device group hierarchy. The shared device group (level 0) is not included in this structure. If the log values are 12, 34, 45, 0, it means that the log was generated by a firewall (or virtual system) that belongs to device group 45, and its ancestors are 34, and 12.

type: keyword


**`panw.panos.timeout`**
:   Timeout after which the IP/User Mappings are cleared.

type: integer


**`panw.panos.vsys_id`**
:   A unique identifier for a virtual system on a Palo Alto Networks firewall.

type: keyword


**`panw.panos.vsys_name`**
:   The name of the virtual system associated with the session; only valid on firewalls enabled for multiple virtual systems.

type: keyword


**`panw.panos.description`**
:   Additional information for any event that has occurred.

type: keyword


**`panw.panos.tunnel_type`**
:   The type of tunnel (either SSLVPN or IPSec).

type: keyword


**`panw.panos.connect_method`**
:   A string showing the how the GlobalProtect app connects to Gateway

type: keyword


**`panw.panos.matchname`**
:   Name of the HIP object or profile.

type: keyword


**`panw.panos.matchtype`**
:   Whether the hip field represents a HIP object or a HIP profile.

type: keyword


**`panw.panos.priority`**
:   The priority order of the gateway that is based on highest (1), high (2), medium (3), low (4), or lowest (5) to which the GlobalProtect app can connect.

type: keyword


**`panw.panos.response_time`**
:   The SSL response time of the selected gateway that is measured in milliseconds on the endpoint during tunnel setup.

type: keyword


**`panw.panos.attempted_gateways`**
:   The fields that are collected for each gateway connection attempt with the gateway name, SSL response time, and priority

type: keyword


**`panw.panos.gateway`**
:   The name of the gateway that is specified on the portal configuration.

type: keyword


**`panw.panos.selection_type`**
:   The connection method that is selected to connect to the gateway.

type: keyword


