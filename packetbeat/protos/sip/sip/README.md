# SIP(Session Initiation Protocol) for packetbeat
The SIP(Session Initiation Protocol) is a communications protocol for signaling and controlling multimedia communication sessions. SIP is used many VoIP applications at not only enterprise uses but also telecom careers.

SIP is text-base protocol like HTTP. But SIP has various unique features like :
- SIP is server-client model, but it roles may changes call by call.
- SIP is request-response model, but server may (usualy) reply many responses for one request.
- There many requests and responses in one call.
- It is not know when the call will end.

## Implementation

### Published for each SIP message(request or response)
- SIP is not a one to one message with request and response. Also order to each message is not determined(a response may be sent after previous response).
- Therefore the SIP response and SIP request is published when packetbeat received the message immidiatory.
- If you need all SIP messages in throughout of SIP dialog, you need to retrieve from Elasticsearch using the SIP Call-ID field etc.

### TCP
* ``transport=tcp`` is not supported yet.

## Configuration

```yaml
- type: sip
  # Configure the ports where to listen for SIP traffic. You can disable
  # the SIP protocol by commenting out the list of ports.
  ports: [5060]

  # Contain parsed SIP headers(defualt true)
  include_headers: true 
  
  # Contain parsed SIP body(defualt true)
  include_body: true

  # Contain raw SIP message(sip header and body,defualt true)
  include_raw: true

  # SIP headers more parse detail(default false)
  parse_detail: false

  # SIP header which is targeted to detail parse are pre-defined (default true)
  #  - As SIP-URI or Name-addr: [From, To, Contact, Record-Route, P-Asserted-Identity, P-Preferred-Identity]
  #  - As Integer: [RSeq, Content-Length, Max-Forwards, Expires, Session-Expires, Min-SE]
  use_default_headers: true

  # Define headers that to parse as SIP-URI or Name-Addr yourself in parse detail mode
  parse_as_uri_for:
    - X-Your-Original-URI
    - X-Your-Addtional-URI

  # Define headers that to parse as Integer yourself in parse detail mode
  parse_as_int_for:
    - X-Your-Original-Number
    - X-Your-Addtional-URI
```


## Output/Published data

### Additional timestamp
- Default timestamp field(@timestamp) precision is not sufficient(the sip response is often send immediately when request received eg. 100 Trying).
- You can sort to keep the message correct order using the ``sip.unixtimenano``(int64) field.

### Request-Line,Status-Line
- In case of SIP request received, stored ``sip.method``(eg.INVITE,BYE,ACK,PRACK) and ``sip.request-uri``.
- In case of SIP response received, stored ``sip.status-code``(eg.200,404) and ``sip.status-phrase``(eg. OK, Ringing)

### Mandatory headers
- SIP mandatory headers From,To,Call-ID.CSeq are stored in ``sip.from``,``sip.to``,``sip.call-id``,``sip.cseq`` fields.

### SIP Headers
- Option ``include_headers`` : If you select true(default true), the outputed JSON contain parsed SIP headers, see below sample's ``sip.headers`` field.
- A SIP header might be exsisted multiple lines(eg. Via). The description order of the SIP header has a meaning. Each SIP header is sotred as dict and the dict has header values as array.
- Compact form will convert and process as longer form.(``t: <sip:foo@example.com`` stored ``{"sip.headers.to": "<sip:foo@example.com>"}``)

### SIP Body
- Option ``include_body`` : If you select true(default true), the outputed JSON contain parsed SIP body, see below sample's ``sip.body`` field.
- SIP allowed having mulitple type of body.
- *Currently it only supports sdp*

### Raw message
- Option ``include_raw`` : If you select true(default true), the outputed JSON contain raw SIP message(sip header and body), see below sample's ``sip.raw`` field.
- Recived raw message is stored in ``raw`` field as text value.

### parse detail mode
- Option ``parse_detail`` : If you select true(default false), the outputed parsed SIP headers are more parse detail like contained SIP-URI, Name-addrs and Integer etc header field, see below **Sample Full JSON Output : Parse Detail Mode** section.
- Option ``use_default_headers`` : If you select true(default true), SIP headers [From, To, Contact, Record-Route, P-Asserted-Identity, P-Preferred-Identity] are parsed detail as SIP-URI or Name-addr, SIP headers [RSeq, Content-Length, Max-Forwards, Expires, Session-Expires, Min-SE] are parse detail as Integer. If you select false, only parse SIP hdeaders [CSeq, Rack].
- SIP headers and Requst-URI will be parsed more detail when ``parse_detail`` parameter set ``true`` in ``packetbeat.yml`` at ``sip`` directive.
- You can parse any SIP headers using below option **Addtional/Cusotm parse target**.

#### Addtional/Custom parse target
- Option ``parse_as_uri_for`` : If you describe header names as list, you can add parse detail header target as SIP-URI or Name-addr.
- Option ``parse_as_int_for`` : If you describe header names as list, you can add parse detail header target as Integer.

#### example case from(to)
 - input>> From: "user"<sip:0312341234@bob.com:5060;transport=udp>;tag=zxcvb;otheroption
 - output>
```
{
    "sip.from":"\"user\"<sip:0312341234@bob.com:5060;transport=udp>;tag=zxcvb;otheroption",
    "sip.headers.from.raw":"\"user\"<sip:0312341234@bob.com:5060;transport=udp>;tag=zxcvb;otheroption",
    "sip.headers.from.display":"user",
    "sip.headers.from.user":"0312341234",
    "sip.headers.from.host":"bob.com",
    "sip.headers.from.port":5060,
    "sip.headers.from.param":["tag=zxcvb","otheroption"]
    "sip.headers.from.uri-param":["transport=udp"]
}
```

#### example case cseq
 - input>> CSeq: 1 INVITE 
 - output>
```
{
    "sip.cseq":"1 INVITE",
    "sip.headers.cseq.raw":"1 INVITE",
    "sip.headers.cseq.number":1,
    "sip.headers.cseq.method":"INVITE"
}
```

#### example case request-uri
 - input>> INVITE sip:9012341234;rn=9012340000;npdi=yes@hoge.com:5060;transport=udp;user=phone SIP/2.0
 - output>
```
{
    "sip.request-uri":"sip:9012341234;rn=9012340000;npdi=yes@hoge.com:5060;transport=udp;user=phone"
    "sip.request-uri-user":"9012341234;rn=9012340000;npdi=yes",
    "sip.request-uri-host":"hoge.com",
    "sip.request-uri-port":"5060",
    "sip.request-uri-params":["transport=udp","user=phone"]
}
```
 
#### example case request-uri(telephone-subscriber)
 - input>> INVITE tel:+819012341234;phone-context=+1234;vnd.company.option=foo SIP/2.0
 - output>
```
{
    "sip.request-uri":"tel:+819012341234;phone-context=+1234;vnd.company.option=foo"
    "sip.request-uri-host":"+819012341234",
    "sip.request-uri-params":["phone-context=+1234","vnd.company.option=foo"]
}
```


### Sample Full JSON Output : Normal Mode

```json
{
   "_index": "packetbeat-7.0.0-alpha1-2018.01.17",
   "_type": "doc",
   "_id": "14uKBGEBLUdHmvOi5U1L",
   "_score": null,
   "_source": {
     "@timestamp": "2018-01-17T14:34:26.016Z",
     "beat": {
       "name": "Elasticsearch1",
       "hostname": "Elasticsearch1",
       "version": "7.0.0-alpha1"
     },
     "sip.headers": {
       "from": [
         "sipp <sip:sipp@192.168.0.220:5060>;tag=26730SIPpTag003138"
       ],
       "to": [
         "service <sip:service@127.0.0.1:5060>"
       ],
       "cseq": [
         "1 INVITE"
       ],
       "subject": [
         "Performance Test"
       ],
       "contact": [
         "sip:sipp@192.168.0.220:5060"
       ],
       "content-type": [
         "application/sdp"
       ],
       "call-id": [
         "3138-26730@192.168.0.220"
       ],
       "content-length": [
         "137"
       ],
       "via": [
         "SIP/2.0/UDP 192.168.0.220:5060;branch=z9hG4bK-26730-3138-0"
       ],
       "max-forwards": [
         "70"
       ]
     },
     "sip.body": {
       "application/sdp": {
         "o": [
           "user1 53655765 2353687637 IN IP4 192.168.0.220"
         ],
         "s": [
           "-"
         ],
         "c": [
           "IN IP4 192.168.0.220"
         ],
         "t": [
           "0 0"
         ],
         "m": [
           "audio 6000 RTP/AVP 0"
         ],
         "a": [
           "rtpmap:0 PCMU/8000"
         ],
         "v": [
           "0"
         ]
       }
     },
     "sip.request-uri": "sip:service@127.0.0.1:5060",
     "sip.call-id": "3138-26730@192.168.0.220",
     "sip.cseq": "1 INVITE",
     "sip.dst": "127.0.0.1:5060",
     "sip.unixtimenano": 1516199666016756000,
     "type": "sip",
     "sip.method": "INVITE",
     "sip.from": "sipp <sip:sipp@192.168.0.220:5060>;tag=26730SIPpTag003138",
     "sip.to": "service <sip:service@127.0.0.1:5060>",
     "sip.raw": """
INVITE sip:service@127.0.0.1:5060 SIP/2.0
Via: SIP/2.0/UDP 192.168.0.220:5060;branch=z9hG4bK-26730-3138-0
From: sipp <sip:sipp@192.168.0.220:5060>;tag=26730SIPpTag003138
To: service <sip:service@127.0.0.1:5060>
Call-ID: 3138-26730@192.168.0.220
CSeq: 1 INVITE
Contact: sip:sipp@192.168.0.220:5060
Max-Forwards: 70
Subject: Performance Test
Content-Type: application/sdp
Content-Length:   137

v=0
o=user1 53655765 2353687637 IN IP4 192.168.0.220
s=-
c=IN IP4 192.168.0.220
t=0 0
m=audio 6000 RTP/AVP 0
a=rtpmap:0 PCMU/8000

""",
    "sip.src": "192.168.0.220:5060",
    "sip.transport": "udp"
  }
}
```

### Sample full JSON Output : Parse Detail Mode

```json
{
  "_id": "vTw21GIB6soIUJW6xglP",
  "_index": "packetbeat-7.0.0-alpha1-2018.04.18",
  "_source": {
    "@timestamp": "2018-04-17T15:26:33.217Z",
    "beat": {
      "hostname": "Elasticsearch1",
      "name": "Elasticsearch1",
      "version": "7.0.0-alpha1"
    },
    "sip.body": {
      "application/sdp": {
        "a": [
          "rtpmap:0 PCMU/8000"
        ],
        "c": [
          "IN IP4 127.0.1.1"
        ],
        "m": [
          "audio 6000 RTP/AVP 0"
        ],
        "o": [
          "user1 53655765 2353687637 IN IP4 127.0.1.1"
        ],
        "s": [
          "-"
        ],
        "t": [
          "0 0"
        ],
        "v": [
          "0"
        ]
      }
    },
    "sip.call-id": "1-30404@127.0.1.1",
    "sip.cseq": "1 INVITE",
    "sip.dst": "192.168.0.23:5060",
    "sip.from": "sipp <sip:sipp@127.0.1.1:5060>;tag=30404SIPpTag001",
    "sip.headers": {
      "call-id": [
        {
          "raw": "1-30404@127.0.1.1"
        }
      ],
      "contact": [
        {
          "host": "127.0.1.1",
          "port": 5060,
          "raw": "sip:sipp@127.0.1.1:5060",
          "user": "sipp"
        }
      ],
      "content-length": [
        {
          "number": 129,
          "raw": "129"
        }
      ],
      "content-type": [
        {
          "raw": "application/sdp"
        }
      ],
      "cseq": [
        {
          "method": "INVITE",
          "number": 1,
          "raw": "1 INVITE"
        }
      ],
      "from": [
        {
          "display": "sipp",
          "host": "127.0.1.1",
          "params": [
            "tag=30404SIPpTag001"
          ],
          "port": 5060,
          "raw": "sipp <sip:sipp@127.0.1.1:5060>;tag=30404SIPpTag001",
          "user": "sipp"
        }
      ],
      "max-forwards": [
        {
          "number": 70,
          "raw": "70"
        }
      ],
      "subject": [
        {
          "raw": "Performance Test"
        }
      ],
      "to": [
        {
          "display": "service",
          "host": "192.168.0.23",
          "port": 5060,
          "raw": "service <sip:service@192.168.0.23:5060>",
          "user": "service"
        }
      ],
      "via": [
        {
          "raw": "SIP/2.0/UDP 127.0.1.1:5060;branch=z9hG4bK-30404-1-0"
        }
      ]
    },
    "sip.method": "INVITE",
    "sip.raw" : """
INVITE sip:service@192.168.0.23:5060 SIP/2.0
Via: SIP/2.0/UDP 127.0.1.1:5060;branch=z9hG4bK-30404-1-0
From: sipp <sip:sipp@127.0.1.1:5060>;tag=30404SIPpTag001
To: service <sip:service@192.168.0.23:5060>
Call-ID: 1-30404@127.0.1.1
CSeq: 1 INVITE
Contact: sip:sipp@127.0.1.1:5060
Max-Forwards: 70
Subject: Performance Test
Content-Type: application/sdp
Content-Length:   129

v=0
o=user1 53655765 2353687637 IN IP4 127.0.1.1
s=-
c=IN IP4 127.0.1.1
t=0 0
m=audio 6000 RTP/AVP 0
a=rtpmap:0 PCMU/8000 

""",
    "sip.request-uri": "sip:service@192.168.0.23:5060",
    "sip.request-uri-host": "192.168.0.23",
    "sip.request-uri-port": 5060,
    "sip.request-uri-user": "service",
    "sip.src": "192.168.0.17:5060",
    "sip.to": "service <sip:service@192.168.0.23:5060>",
    "sip.transport": "udp",
    "sip.unixtimenano": 1523978793217683000,
    "type": "sip"
  },
  "_type": "doc",
  "_version": 1,
  "found": true
}
```

## TODO
* In case of body was encoded, Content-encode
* SIP/TCP
* More SIP content support.
 - ISUP(SIP-I/SIP-T)
 - multipart/form-data boundary

