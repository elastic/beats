---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/exported-fields-flows_event.html
---

# Flow Event fields [exported-fields-flows_event]

These fields contain data about the flow itself.

**`flow.final`**
:   Indicates if event is last event in flow. If final is false, the event reports an intermediate flow state only.

type: boolean


**`flow.id`**
:   Internal flow ID based on connection meta data and address.


**`flow.vlan`**
:   VLAN identifier from the 802.1q frame. In case of a multi-tagged frame this field will be an array with the outer tag’s VLAN identifier listed first.

type: long


**`flow_id`**
:   type: alias

alias to: flow.id


**`final`**
:   type: alias

alias to: flow.final


**`vlan`**
:   type: alias

alias to: flow.vlan


**`source.stats.net_bytes_total`**
:   type: alias

alias to: source.bytes


**`source.stats.net_packets_total`**
:   type: alias

alias to: source.packets


**`dest.stats.net_bytes_total`**
:   type: alias

alias to: destination.bytes


**`dest.stats.net_packets_total`**
:   type: alias

alias to: destination.packets


