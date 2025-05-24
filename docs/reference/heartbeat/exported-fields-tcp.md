---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/Heartbeat/current/exported-fields-tcp.html
---

# TCP layer fields [exported-fields-tcp]

None


## tcp [_tcp]

TCP network layer related fields.


**`tcp.port`**
:   Service port number.

type: alias

alias to: url.port


## rtt [_rtt]

TCP layer round trip times.


## connect [_connect]

Duration required to establish a TCP connection based on already available IP address.


**`tcp.rtt.connect.us`**
:   Duration in microseconds

type: long


## validate [_validate]

Duration of validation step based on existing TCP connection.


**`tcp.rtt.validate.us`**
:   Duration in microseconds

type: long


