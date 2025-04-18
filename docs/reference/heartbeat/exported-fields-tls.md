---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/exported-fields-tls.html
---

# TLS encryption layer fields [exported-fields-tls]

None


## tls [_tls_2]

TLS layer related fields.

**`tls.certificate_not_valid_before`**
:   [7.8.0]

Deprecated in favor of `tls.server.x509.not_before`. Earliest time at which the connection’s certificates are valid.

type: date


**`tls.certificate_not_valid_after`**
:   [7.8.0]

Deprecated in favor of `tls.server.x509.not_after`. Latest time at which the connection’s certificates are valid.

type: date



## rtt [_rtt_6]

TLS layer round trip times.


## handshake [_handshake]

Time required to finish TLS handshake based on already available network connection.

**`tls.rtt.handshake.us`**
:   Duration in microseconds

type: long



## server [_server_2]

Detailed x509 certificate metadata

**`tls.server.version_number`**
:   Version of x509 format.

type: keyword

example: 3


