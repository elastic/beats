---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-apache.html
---

# Apache fields [exported-fields-apache]

Apache HTTPD server metricsets collected from the Apache web server.


## apache [_apache]

`apache` contains the metrics that were scraped from Apache.


## status [_status]

`status` contains the metrics that were scraped from the Apache status page.

**`apache.status.hostname`**
:   Apache hostname.

type: keyword


**`apache.status.total_accesses`**
:   Total number of access requests.

type: long


**`apache.status.total_kbytes`**
:   Total number of kilobytes served.

type: long


**`apache.status.requests_per_sec`**
:   Requests per second.

type: scaled_float


**`apache.status.bytes_per_sec`**
:   Bytes per second.

type: scaled_float


**`apache.status.bytes_per_request`**
:   Bytes per request.

type: scaled_float


**`apache.status.workers.busy`**
:   Number of busy workers.

type: long


**`apache.status.workers.idle`**
:   Number of idle workers.

type: long



## uptime [_uptime_2]

Uptime stats.

**`apache.status.uptime.server_uptime`**
:   Server uptime in seconds.

type: long


**`apache.status.uptime.uptime`**
:   Server uptime.

type: long



## cpu [_cpu_2]

CPU stats.

**`apache.status.cpu.load`**
:   CPU Load.

type: scaled_float


**`apache.status.cpu.user`**
:   CPU user load.

type: scaled_float


**`apache.status.cpu.system`**
:   System cpu.

type: scaled_float


**`apache.status.cpu.children_user`**
:   CPU of children user.

type: scaled_float


**`apache.status.cpu.children_system`**
:   CPU of children system.

type: scaled_float



## connections [_connections]

Connection stats.

**`apache.status.connections.total`**
:   Total connections.

type: long


**`apache.status.connections.async.writing`**
:   Async connection writing.

type: long


**`apache.status.connections.async.keep_alive`**
:   Async keeped alive connections.

type: long


**`apache.status.connections.async.closing`**
:   Async closed connections.

type: long



## load [_load_2]

Load averages.

**`apache.status.load.1`**
:   Load average for the last minute.

type: scaled_float


**`apache.status.load.5`**
:   Load average for the last 5 minutes.

type: scaled_float


**`apache.status.load.15`**
:   Load average for the last 15 minutes.

type: scaled_float



## scoreboard [_scoreboard]

Scoreboard metrics.

**`apache.status.scoreboard.starting_up`**
:   Starting up.

type: long


**`apache.status.scoreboard.reading_request`**
:   Reading requests.

type: long


**`apache.status.scoreboard.sending_reply`**
:   Sending Reply.

type: long


**`apache.status.scoreboard.keepalive`**
:   Keep alive.

type: long


**`apache.status.scoreboard.dns_lookup`**
:   Dns Lookups.

type: long


**`apache.status.scoreboard.closing_connection`**
:   Closing connections.

type: long


**`apache.status.scoreboard.logging`**
:   Logging

type: long


**`apache.status.scoreboard.gracefully_finishing`**
:   Gracefully finishing.

type: long


**`apache.status.scoreboard.idle_cleanup`**
:   Idle cleanups.

type: long


**`apache.status.scoreboard.open_slot`**
:   Open slots.

type: long


**`apache.status.scoreboard.waiting_for_connection`**
:   Waiting for connections.

type: long


**`apache.status.scoreboard.total`**
:   Total.

type: long


