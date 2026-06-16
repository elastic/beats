---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/exported-fields-resolve.html
applies_to:
  stack: ga
  serverless: ga
---

% This file is generated! See dev-tools/mage/generate_fields_docs.go

# Host lookup fields [exported-fields-resolve]

None

## resolve [_resolve]

Host lookup fields.

**`resolve.host`**
:   Hostname of service being monitored.

    type: alias

    alias to: url.domain


**`resolve.ip`**
:   IP address found for the given host.

    type: ip


## rtt [_rtt]

Duration required to resolve an IP from hostname.

**`resolve.rtt.us`**
:   Duration in microseconds

    type: long


