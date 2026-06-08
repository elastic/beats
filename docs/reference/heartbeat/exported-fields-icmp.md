---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/exported-fields-icmp.html
applies_to:
  stack: ga
  serverless: ga
---

% This file is generated! See dev-tools/mage/generate_fields_docs.go

# ICMP fields [exported-fields-icmp]

None

## icmp [_icmp]

IP ping fields.

**`icmp.requests`**
:   Number if ICMP EchoRequests send.

    type: integer


## rtt [_rtt]

ICMP Echo Request and Reply round trip time

**`icmp.rtt.us`**
:   Duration in microseconds

    type: long


