---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/exported-fields-resolve.html
---

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



## rtt [_rtt_3]

Duration required to resolve an IP from hostname.

**`resolve.rtt.us`**
:   Duration in microseconds

type: long


