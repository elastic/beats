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
* ``content-encoding`` is not supported yet.
* Default timestamp field(@timestamp) precision is not sufficient(the sip response is often send immediately when request received eg. 100 Trying). You can sort to keep the message correct order using the ``sip.timestamp``(`date_nanos`) field.
* Body parsing is partially supported for ``application/sdp`` content type only.

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

  # You can monitor tcp SIP traffic by setting the transport_protocol option
  # to tcp, it defaults to udp.
  #transport_protocol: tcp
```

### Sample Full JSON Output

```json
{
    "@metadata.beat": "packetbeat",
    "@metadata.type": "_doc",
    "client.ip": "192.168.1.2",
    "client.port": 5060,
    "destination.ip": "212.242.33.35",
    "destination.port": 5060,
    "event.action": "sip_register",
    "event.category": [
        "network",
        "authentication"
    ],
    "event.dataset": "sip",
    "event.duration": 0,
    "event.kind": "event",
    "event.original": "REGISTER sip:sip.cybercity.dk SIP/2.0\r\nVia: SIP/2.0/UDP 192.168.1.2;branch=z9hG4bKnp112903503-43a64480192.168.1.2;rport\r\nFrom: <sip:voi18062@sip.cybercity.dk>;tag=6bac55c\r\nTo: <sip:voi18062@sip.cybercity.dk>\r\nCall-ID: 578222729-4665d775@578222732-4665d772\r\nContact:  <sip:voi18062@192.168.1.2:5060;line=aca6b97ca3f5e51a>;expires=1200;q=0.500\r\nExpires: 1200\r\nCSeq: 75 REGISTER\r\nContent-Length: 0\r\nAuthorization: Digest username=\"voi18062\",realm=\"sip.cybercity.dk\",uri=\"sip:192.168.1.2\",nonce=\"1701b22972b90f440c3e4eb250842bb\",opaque=\"1701a1351f70795\",nc=\"00000001\",response=\"79a0543188495d288c9ebbe0c881abdc\"\r\nMax-Forwards: 70\r\nUser-Agent: Nero SIPPS IP Phone Version 2.0.51.16\r\n\r\n",
    "event.sequence": 75,
    "event.type": [
        "info",
        "protocol"
    ],
    "network.application": "sip",
    "network.community_id": "1:dOa61R2NaaJsJlcFAiMIiyXX+Kk=",
    "network.iana_number": "17",
    "network.protocol": "sip",
    "network.transport": "udp",
    "network.type": "ipv4",
    "related.hosts": [
        "sip.cybercity.dk"
    ],
    "related.ip": [
        "192.168.1.2",
        "212.242.33.35"
    ],
    "related.user": [
        "voi18062"
    ],
    "server.ip": "212.242.33.35",
    "server.port": 5060,
    "sip.auth.realm": "sip.cybercity.dk",
    "sip.auth.scheme": "Digest",
    "sip.auth.uri.host": "192.168.1.2",
    "sip.auth.uri.original": "sip:192.168.1.2",
    "sip.auth.uri.scheme": "sip",
    "sip.call_id": "578222729-4665d775@578222732-4665d772",
    "sip.contact.uri.host": "sip.cybercity.dk",
    "sip.contact.uri.original": "sip:voi18062@sip.cybercity.dk",
    "sip.contact.uri.scheme": "sip",
    "sip.contact.uri.username": "voi18062",
    "sip.cseq.code": 75,
    "sip.cseq.method": "REGISTER",
    "sip.from.tag": "6bac55c",
    "sip.from.uri.host": "sip.cybercity.dk",
    "sip.from.uri.original": "sip:voi18062@sip.cybercity.dk",
    "sip.from.uri.scheme": "sip",
    "sip.from.uri.username": "voi18062",
    "sip.max_forwards": 70,
    "sip.method": "REGISTER",
    "sip.to.uri.host": "sip.cybercity.dk",
    "sip.to.uri.original": "sip:voi18062@sip.cybercity.dk",
    "sip.to.uri.scheme": "sip",
    "sip.to.uri.username": "voi18062",
    "sip.type": "request",
    "sip.uri.host": "sip.cybercity.dk",
    "sip.uri.original": "sip:sip.cybercity.dk",
    "sip.uri.scheme": "sip",
    "sip.user_agent.original": "Nero SIPPS IP Phone Version 2.0.51.16",
    "sip.version": "2.0",
    "sip.via.original": [
        "SIP/2.0/UDP 192.168.1.2;branch=z9hG4bKnp112903503-43a64480192.168.1.2;rport"
    ],
    "source.ip": "192.168.1.2",
    "source.port": 5060,
    "status": "OK",
    "type": "sip",
    "user.name": "voi18062"
}
```

