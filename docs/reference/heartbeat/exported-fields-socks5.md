---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/exported-fields-socks5.html
applies_to:
  stack: ga
  serverless: ga
---

% This file is generated! See dev-tools/mage/generate_fields_docs.go

# SOCKS5 proxy fields [exported-fields-socks5]

None

## socks5 [_socks5]

SOCKS5 proxy related fields:

## rtt [_rtt]

TLS layer round trip times.

## connect [_connect]

Time required to establish a connection via SOCKS5 to endpoint based on available connection to SOCKS5 proxy.

**`socks5.rtt.connect.us`**
:   Duration in microseconds

    type: long


