---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/exported-fields-dhcpv4.html
---

# DHCPv4 fields [exported-fields-dhcpv4]

DHCPv4 event fields

**`dhcpv4.transaction_id`**
:   Transaction ID, a random number chosen by the client, used by the client and server to associate messages and responses between a client and a server.

type: keyword


**`dhcpv4.seconds`**
:   Number of seconds elapsed since client began address acquisition or renewal process.

type: long


**`dhcpv4.flags`**
:   Flags are set by the client to indicate how the DHCP server should its reply — either unicast or broadcast.

type: keyword


**`dhcpv4.client_ip`**
:   The current IP address of the client.

type: ip


**`dhcpv4.assigned_ip`**
:   The IP address that the DHCP server is assigning to the client. This field is also known as "your" IP address.

type: ip


**`dhcpv4.server_ip`**
:   The IP address of the DHCP server that the client should use for the next step in the bootstrap process.

type: ip


**`dhcpv4.relay_ip`**
:   The relay IP address used by the client to contact the server (i.e. a DHCP relay server).

type: ip


**`dhcpv4.client_mac`**
:   The client’s MAC address (layer two).

type: keyword


**`dhcpv4.server_name`**
:   The name of the server sending the message. Optional. Used in DHCPOFFER or DHCPACK messages.

type: keyword


**`dhcpv4.op_code`**
:   The message op code (bootrequest or bootreply).

type: keyword

example: bootreply


**`dhcpv4.hops`**
:   The number of hops the DHCP message went through.

type: long


**`dhcpv4.hardware_type`**
:   The type of hardware used for the local network (Ethernet, LocalTalk, etc).

type: keyword


**`dhcpv4.option.message_type`**
:   The specific type of DHCP message being sent (e.g. discover, offer, request, decline, ack, nak, release, inform).

type: keyword

example: ack


**`dhcpv4.option.parameter_request_list`**
:   This option is used by a DHCP client to request values for specified configuration parameters.

type: keyword


**`dhcpv4.option.requested_ip_address`**
:   This option is used in a client request (DHCPDISCOVER) to allow the client to request that a particular IP address be assigned.

type: ip


**`dhcpv4.option.server_identifier`**
:   IP address of the individual DHCP server which handled this message.

type: ip


**`dhcpv4.option.broadcast_address`**
:   This option specifies the broadcast address in use on the client’s subnet.

type: ip


**`dhcpv4.option.max_dhcp_message_size`**
:   This option specifies the maximum length DHCP message that the client is willing to accept.

type: long


**`dhcpv4.option.class_identifier`**
:   This option is used by DHCP clients to optionally identify the vendor type and configuration of a DHCP client. Vendors may choose to define specific vendor class identifiers to convey particular configuration or other identification information about a client.  For example, the identifier may encode the client’s hardware configuration.

type: keyword


**`dhcpv4.option.domain_name`**
:   This option specifies the domain name that client should use when resolving hostnames via the Domain Name System.

type: keyword


**`dhcpv4.option.dns_servers`**
:   The domain name server option specifies a list of Domain Name System servers available to the client.

type: ip


**`dhcpv4.option.vendor_identifying_options`**
:   A DHCP client may use this option to unambiguously identify the vendor that manufactured the hardware on which the client is running, the software in use, or an industry consortium to which the vendor belongs. This field is described in RFC 3925.

type: object


**`dhcpv4.option.subnet_mask`**
:   The subnet mask that the client should use on the currnet network.

type: ip


**`dhcpv4.option.utc_time_offset_sec`**
:   The time offset field specifies the offset of the client’s subnet in seconds from Coordinated Universal Time (UTC).

type: long


**`dhcpv4.option.router`**
:   The router option specifies a list of IP addresses for routers on the client’s subnet.

type: ip


**`dhcpv4.option.time_servers`**
:   The time server option specifies a list of RFC 868 time servers available to the client.

type: ip


**`dhcpv4.option.ntp_servers`**
:   This option specifies a list of IP addresses indicating NTP servers available to the client.

type: ip


**`dhcpv4.option.hostname`**
:   This option specifies the name of the client.

type: keyword


**`dhcpv4.option.ip_address_lease_time_sec`**
:   This option is used in a client request (DHCPDISCOVER or DHCPREQUEST) to allow the client to request a lease time for the IP address.  In a server reply (DHCPOFFER), a DHCP server uses this option to specify the lease time it is willing to offer.

type: long


**`dhcpv4.option.message`**
:   This option is used by a DHCP server to provide an error message to a DHCP client in a DHCPNAK message in the event of a failure. A client may use this option in a DHCPDECLINE message to indicate the why the client declined the offered parameters.

type: text


**`dhcpv4.option.renewal_time_sec`**
:   This option specifies the time interval from address assignment until the client transitions to the RENEWING state.

type: long


**`dhcpv4.option.rebinding_time_sec`**
:   This option specifies the time interval from address assignment until the client transitions to the REBINDING state.

type: long


**`dhcpv4.option.boot_file_name`**
:   This option is used to identify a bootfile when the *file* field in the DHCP header has been used for DHCP options.

type: keyword


