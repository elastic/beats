---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-iptables.html
---

# iptables fields [exported-fields-iptables]

Module for handling the iptables logs.


## iptables [_iptables]

Fields from the iptables logs.

**`iptables.ether_type`**
:   Value of the ethernet type field identifying the network layer protocol.

type: long


**`iptables.flow_label`**
:   IPv6 flow label.

type: integer


**`iptables.fragment_flags`**
:   IP fragment flags. A combination of CE, DF and MF.

type: keyword


**`iptables.fragment_offset`**
:   Offset of the current IP fragment.

type: long



## icmp [_icmp]

ICMP fields.

**`iptables.icmp.code`**
:   ICMP code.

type: long


**`iptables.icmp.id`**
:   ICMP ID.

type: long


**`iptables.icmp.parameter`**
:   ICMP parameter.

type: long


**`iptables.icmp.redirect`**
:   ICMP redirect address.

type: ip


**`iptables.icmp.seq`**
:   ICMP sequence number.

type: long


**`iptables.icmp.type`**
:   ICMP type.

type: long


**`iptables.id`**
:   Packet identifier.

type: long


**`iptables.incomplete_bytes`**
:   Number of incomplete bytes.

type: long


**`iptables.input_device`**
:   Device that received the packet.

type: keyword


**`iptables.precedence_bits`**
:   IP precedence bits.

type: short


**`iptables.tos`**
:   IP Type of Service field.

type: long


**`iptables.length`**
:   Packet length.

type: long


**`iptables.output_device`**
:   Device that output the packet.

type: keyword



## tcp [_tcp_2]

TCP fields.

**`iptables.tcp.flags`**
:   TCP flags.

type: keyword


**`iptables.tcp.reserved_bits`**
:   TCP reserved bits.

type: short


**`iptables.tcp.seq`**
:   TCP sequence number.

type: long


**`iptables.tcp.ack`**
:   TCP Acknowledgment number.

type: long


**`iptables.tcp.window`**
:   Advertised TCP window size.

type: long


**`iptables.ttl`**
:   Time To Live field.

type: integer



## udp [_udp]

UDP fields.

**`iptables.udp.length`**
:   Length of the UDP header and payload.

type: long



## ubiquiti [_ubiquiti]

Fields for Ubiquiti network devices.

**`iptables.ubiquiti.input_zone`**
:   Input zone.

type: keyword


**`iptables.ubiquiti.output_zone`**
:   Output zone.

type: keyword


**`iptables.ubiquiti.rule_number`**
:   The rule number within the rule set.

type: keyword


**`iptables.ubiquiti.rule_set`**
:   The rule set name.

type: keyword


