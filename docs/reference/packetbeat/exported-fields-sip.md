---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/exported-fields-sip.html
---

# SIP fields [exported-fields-sip]

SIP-specific event fields.


## sip [_sip]

Information about SIP traffic.

**`sip.code`**
:   Response status code.

type: keyword


**`sip.method`**
:   Request method.

type: keyword


**`sip.status`**
:   Response status phrase.

type: keyword


**`sip.type`**
:   Either request or response.

type: keyword


**`sip.version`**
:   SIP protocol version.

type: keyword


**`sip.uri.original`**
:   The original URI.

type: keyword


**`sip.uri.original.text`**
:   type: text


**`sip.uri.scheme`**
:   The URI scheme.

type: keyword


**`sip.uri.username`**
:   The URI user name.

type: keyword


**`sip.uri.host`**
:   The URI host.

type: keyword


**`sip.uri.port`**
:   The URI port.

type: keyword


**`sip.accept`**
:   Accept header value.

type: keyword


**`sip.allow`**
:   Allowed methods.

type: keyword


**`sip.call_id`**
:   Call ID.

type: keyword


**`sip.content_length`**
:   type: long


**`sip.content_type`**
:   type: keyword


**`sip.max_forwards`**
:   type: long


**`sip.supported`**
:   Supported methods.

type: keyword


**`sip.user_agent.original`**
:   type: keyword


**`sip.user_agent.original.text`**
:   type: text


**`sip.private.uri.original`**
:   Private original URI.

type: keyword


**`sip.private.uri.original.text`**
:   type: text


**`sip.private.uri.scheme`**
:   Private URI scheme.

type: keyword


**`sip.private.uri.username`**
:   Private URI user name.

type: keyword


**`sip.private.uri.host`**
:   Private URI host.

type: keyword


**`sip.private.uri.port`**
:   Private URI port.

type: keyword


**`sip.cseq.code`**
:   Sequence code.

type: keyword


**`sip.cseq.method`**
:   Sequence method.

type: keyword


**`sip.via.original`**
:   The original Via value.

type: keyword


**`sip.via.original.text`**
:   type: text


**`sip.to.display_info`**
:   To display info

type: keyword


**`sip.to.uri.original`**
:   To original URI

type: keyword


**`sip.to.uri.original.text`**
:   type: text


**`sip.to.uri.scheme`**
:   To URI scheme

type: keyword


**`sip.to.uri.username`**
:   To URI user name

type: keyword


**`sip.to.uri.host`**
:   To URI host

type: keyword


**`sip.to.uri.port`**
:   To URI port

type: keyword


**`sip.to.tag`**
:   To tag

type: keyword


**`sip.from.display_info`**
:   From display info

type: keyword


**`sip.from.uri.original`**
:   From original URI

type: keyword


**`sip.from.uri.original.text`**
:   type: text


**`sip.from.uri.scheme`**
:   From URI scheme

type: keyword


**`sip.from.uri.username`**
:   From URI user name

type: keyword


**`sip.from.uri.host`**
:   From URI host

type: keyword


**`sip.from.uri.port`**
:   From URI port

type: keyword


**`sip.from.tag`**
:   From tag

type: keyword


**`sip.contact.display_info`**
:   Contact display info

type: keyword


**`sip.contact.uri.original`**
:   Contact original URI

type: keyword


**`sip.contact.uri.original.text`**
:   type: text


**`sip.contact.uri.scheme`**
:   Contat URI scheme

type: keyword


**`sip.contact.uri.username`**
:   Contact URI user name

type: keyword


**`sip.contact.uri.host`**
:   Contact URI host

type: keyword


**`sip.contact.uri.port`**
:   Contact URI port

type: keyword


**`sip.contact.transport`**
:   Contact transport

type: keyword


**`sip.contact.line`**
:   Contact line

type: keyword


**`sip.contact.expires`**
:   Contact expires

type: keyword


**`sip.contact.q`**
:   Contact Q

type: keyword


**`sip.auth.scheme`**
:   Auth scheme

type: keyword


**`sip.auth.realm`**
:   Auth realm

type: keyword


**`sip.auth.uri.original`**
:   Auth original URI

type: keyword


**`sip.auth.uri.original.text`**
:   type: text


**`sip.auth.uri.scheme`**
:   Auth URI scheme

type: keyword


**`sip.auth.uri.host`**
:   Auth URI host

type: keyword


**`sip.auth.uri.port`**
:   Auth URI port

type: keyword


**`sip.sdp.version`**
:   SDP version

type: keyword


**`sip.sdp.owner.username`**
:   SDP owner user name

type: keyword


**`sip.sdp.owner.session_id`**
:   SDP owner session ID

type: keyword


**`sip.sdp.owner.version`**
:   SDP owner version

type: keyword


**`sip.sdp.owner.ip`**
:   SDP owner IP

type: ip


**`sip.sdp.session.name`**
:   SDP session name

type: keyword


**`sip.sdp.connection.info`**
:   SDP connection info

type: keyword


**`sip.sdp.connection.address`**
:   SDP connection address

type: keyword


**`sip.sdp.body.original`**
:   SDP original body

type: keyword


**`sip.sdp.body.original.text`**
:   type: text


