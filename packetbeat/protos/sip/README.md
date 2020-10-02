# SIP (Session Initiation Protocol) for packetbeat

The SIP (Session Initiation Protocol) is a communications protocol for signaling and controlling multimedia communication sessions. SIP is used by many VoIP applications, not only for enterprise uses but also telecom carriers.

SIP is a text-based protocol like HTTP. But SIP has various unique features like :
- SIP is server-client model, but its role may change in a per call basis.
- SIP is request-response model, but server may (usually) reply with many responses to a single request.
- There are many requests and responses in one call.
- It is not known when the call will end.

## Implementation

### Published for each SIP message (request or response)

- SIP is not a one to one message with request and response. Also order to each message is not determined (a response may be sent after previous response).
- Therefore the SIP responses and requests are published when packetbeat receives them immediately.
- If you need all SIP messages in throughout of SIP dialog, you need to retrieve from Elasticsearch using the SIP Call ID field etc.

### Notes
* ``transport=tcp`` is not supported yet.
* Default timestamp field(@timestamp) precision is not sufficient(the sip response is often send immediately when request received eg. 100 Trying). You can sort to keep the message correct order using the ``sip.timestamp``(int64) field.

## Configuration

```yaml
- type: sip
  # Configure the ports where to listen for SIP traffic. You can disable the SIP protocol by commenting out the list of ports.
  ports: [5060]

  # Parse the authorization headers
  parse_authorization: true

  # Parse body contents (only when body is SDP)
  parse_body: true

  # Preserve original contents in event.original
  keep_original: true
```

### Sample Full JSON Output

```json
{
    "@metadata.beat": "packetbeat",
    "@metadata.type": "_doc",
    "client.ip": "10.0.2.20",
    "client.port": 5060,
    "destination.ip": "10.0.2.15",
    "destination.port": 5060,
    "event.action": "sip_invite",
    "event.category": [
        "network",
        "protocol"
    ],
    "event.dataset": "sip",
    "event.duration": 0,
    "event.kind": "event",
    "event.original": "INVITE sip:test@10.0.2.15:5060 SIP/2.0\r\nVia: SIP/2.0/UDP 10.0.2.20:5060;branch=z9hG4bK-2187-1-0\r\nFrom: \"DVI4/8000\" <sip:sipp@10.0.2.20:5060>;tag=1\r\nTo: test <sip:test@10.0.2.15:5060>\r\nCall-ID: 1-2187@10.0.2.20\r\nCSeq: 1 INVITE\r\nContact: sip:sipp@10.0.2.20:5060\r\nMax-Forwards: 70\r\nContent-Type: application/sdp\r\nContent-Length:   123\r\n\r\nv=0\r\no=- 42 42 IN IP4 10.0.2.20\r\ns=-\r\nc=IN IP4 10.0.2.20\r\nt=0 0\r\nm=audio 6000 RTP/AVP 5\r\na=rtpmap:5 DVI4/8000\r\na=recvonly\r\n",
    "event.sequence": 1,
    "event.type": [
        "info"
    ],
    "network.application": "sip",
    "network.community_id": "1:xDRQZvk3ErEhBDslXv1c6EKI804=",
    "network.iana_number": "17",
    "network.protocol": "sip",
    "network.transport": "udp",
    "network.type": "ipv4",
    "related.hosts": [
        "10.0.2.15",
        "10.0.2.20"
    ],
    "related.ip": [
        "10.0.2.20",
        "10.0.2.15"
    ],
    "related.user": [
        "test",
        "sipp"
    ],
    "server.ip": "10.0.2.15",
    "server.port": 5060,
    "sip.call_id": "1-2187@10.0.2.20",
    "sip.content_length": 123,
    "sip.content_type": "application/sdp",
    "sip.cseq.code": 1,
    "sip.cseq.method": "INVITE",
    "sip.from.display_info": "DVI4/8000",
    "sip.from.tag": "1",
    "sip.from.uri.host": "10.0.2.20",
    "sip.from.uri.original": "sip:sipp@10.0.2.20:5060",
    "sip.from.uri.port": 5060,
    "sip.from.uri.scheme": "sip",
    "sip.from.uri.username": "sipp",
    "sip.max_forwards": 70,
    "sip.method": "INVITE",
    "sip.to.display_info": "test",
    "sip.to.uri.host": "10.0.2.15",
    "sip.to.uri.original": "sip:test@10.0.2.15:5060",
    "sip.to.uri.port": 5060,
    "sip.to.uri.scheme": "sip",
    "sip.to.uri.username": "test",
    "sip.type": "request",
    "sip.uri.host": "10.0.2.15",
    "sip.uri.original": "sip:test@10.0.2.15:5060",
    "sip.uri.port": 5060,
    "sip.uri.scheme": "sip",
    "sip.uri.username": "test",
    "sip.version": "2.0",
    "source.ip": "10.0.2.20",
    "source.port": 5060,
    "status": "OK",
    "type": "sip"
}
```

