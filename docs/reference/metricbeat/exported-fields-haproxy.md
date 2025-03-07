---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-haproxy.html
---

# HAProxy fields [exported-fields-haproxy]

HAProxy Module


## haproxy [_haproxy]

HAProxy metrics.


## info [_info_5]

General information about HAProxy processes.

**`haproxy.info.processes`**
:   Number of processes.

type: long


**`haproxy.info.process_num`**
:   Process number.

type: long


**`haproxy.info.threads`**
:   Number of threads.

type: long


**`haproxy.info.pid`**
:   Process ID.

type: alias

alias to: process.pid


**`haproxy.info.run_queue`**
:   type: long


**`haproxy.info.stopping`**
:   Number of stopping jobs.

type: long


**`haproxy.info.jobs`**
:   Number of all jobs.

type: long


**`haproxy.info.unstoppable_jobs`**
:   Number of unstoppable jobs.

type: long


**`haproxy.info.listeners`**
:   Number of listeners.

type: long


**`haproxy.info.dropped_logs`**
:   Number of dropped logs.

type: long


**`haproxy.info.busy_polling`**
:   Number of busy polling.

type: long


**`haproxy.info.failed_resolutions`**
:   Number of failed resolutions.

type: long


**`haproxy.info.tasks`**
:   type: long


**`haproxy.info.uptime.sec`**
:   Current uptime in seconds.

type: long


**`haproxy.info.memory.max.bytes`**
:   Maximum amount of memory usage in bytes (the *Memmax_MB* value converted to bytes).

type: long

format: bytes


**`haproxy.info.bytes.out.total`**
:   Number of bytes sent out.

type: long


**`haproxy.info.bytes.out.rate`**
:   Average bytes output rate.

type: long


**`haproxy.info.peers.active`**
:   Number of active peers.

type: long


**`haproxy.info.peers.connected`**
:   Number of connected peers.

type: long


**`haproxy.info.pool.allocated`**
:   Size of the allocated pool.

type: long


**`haproxy.info.pool.used`**
:   Number of members used from the allocated pool.

type: long


**`haproxy.info.pool.failed`**
:   Number of failed connections to pool members.

type: long


**`haproxy.info.ulimit_n`**
:   Maximum number of open files for the process.

type: long



## compress [_compress]


## bps [_bps]

**`haproxy.info.compress.bps.in`**
:   Incoming compressed data in bits per second.

type: long


**`haproxy.info.compress.bps.out`**
:   Outgoing compressed data in bits per second.

type: long


**`haproxy.info.compress.bps.rate_limit`**
:   Rate limit of compressed data in bits per second.

type: long



## connection [_connection]


## rate [_rate]

**`haproxy.info.connection.rate.value`**
:   Number of connections in the last second.

type: long


**`haproxy.info.connection.rate.limit`**
:   Rate limit of connections.

type: long


**`haproxy.info.connection.rate.max`**
:   Maximum rate of connections.

type: long


**`haproxy.info.connection.current`**
:   Current connections.

type: long


**`haproxy.info.connection.total`**
:   Total connections.

type: long


**`haproxy.info.connection.ssl.current`**
:   Current SSL connections.

type: long


**`haproxy.info.connection.ssl.total`**
:   Total SSL connections.

type: long


**`haproxy.info.connection.ssl.max`**
:   Maximum SSL connections.

type: long


**`haproxy.info.connection.max`**
:   Maximum connections.

type: long


**`haproxy.info.connection.hard_max`**
:   type: long


**`haproxy.info.requests.total`**
:   Total number of requests.

type: long


**`haproxy.info.sockets.max`**
:   Maximum number of sockets.

type: long


**`haproxy.info.requests.max`**
:   Maximum number of requests.

type: long



## pipes [_pipes]

**`haproxy.info.pipes.used`**
:   Number of used pipes during kernel-based tcp splicing.

type: integer


**`haproxy.info.pipes.free`**
:   Number of free pipes.

type: integer


**`haproxy.info.pipes.max`**
:   Maximum number of used pipes.

type: integer



## session [_session]

None

**`haproxy.info.session.rate.value`**
:   Rate of session per seconds.

type: integer


**`haproxy.info.session.rate.limit`**
:   Rate limit of sessions.

type: integer


**`haproxy.info.session.rate.max`**
:   Maximum rate of sessions.

type: integer



## ssl [_ssl_7]

None

**`haproxy.info.ssl.rate.value`**
:   Rate of SSL requests.

type: integer


**`haproxy.info.ssl.rate.limit`**
:   Rate limit of SSL requests.

type: integer


**`haproxy.info.ssl.rate.max`**
:   Maximum rate of SSL requests.

type: integer



## frontend [_frontend]

None

**`haproxy.info.ssl.frontend.key_rate.value`**
:   Key rate of SSL frontend.

type: integer


**`haproxy.info.ssl.frontend.key_rate.max`**
:   Maximum key rate of SSL frontend.

type: integer


**`haproxy.info.ssl.frontend.session_reuse.pct`**
:   Rate of reuse of SSL frontend sessions.

type: scaled_float

format: percent



## backend [_backend]

None

**`haproxy.info.ssl.backend.key_rate.value`**
:   Key rate of SSL backend sessions.

type: integer


**`haproxy.info.ssl.backend.key_rate.max`**
:   Maximum key rate of SSL backend sessions.

type: integer


**`haproxy.info.ssl.cached_lookups`**
:   Number of SSL cache lookups.

type: long


**`haproxy.info.ssl.cache_misses`**
:   Number of SSL cache misses.

type: long



## zlib_mem_usage [_zlib_mem_usage]

**`haproxy.info.zlib_mem_usage.value`**
:   Memory usage of zlib.

type: integer


**`haproxy.info.zlib_mem_usage.max`**
:   Maximum memory usage of zlib.

type: integer


**`haproxy.info.idle.pct`**
:   Percentage of idle time.

type: scaled_float

format: percent



## stat [_stat]

Stats collected from HAProxy processes.

**`haproxy.stat.status`**
:   Status (UP, DOWN, NOLB, MAINT, or MAINT(via)…​).

type: keyword


**`haproxy.stat.weight`**
:   Total weight (for backends), or server weight (for servers).

type: long


**`haproxy.stat.downtime`**
:   Total downtime (in seconds). For backends, this value is the downtime for the whole backend, not the sum of the downtime for the servers.

type: long


**`haproxy.stat.component_type`**
:   Component type (0=frontend, 1=backend, 2=server, or 3=socket/listener).

type: integer


**`haproxy.stat.process_id`**
:   Process ID (0 for first instance, 1 for second, and so on).

type: alias

alias to: process.pid


**`haproxy.stat.service_name`**
:   Service name (FRONTEND for frontend, BACKEND for backend, or any name for server/listener).

type: keyword


**`haproxy.stat.in.bytes`**
:   Bytes in.

type: long

format: bytes


**`haproxy.stat.out.bytes`**
:   Bytes out.

type: long

format: bytes


**`haproxy.stat.last_change`**
:   Number of seconds since the last UP→DOWN or DOWN→UP transition.

type: integer


**`haproxy.stat.throttle.pct`**
:   Current throttle percentage for the server when slowstart is active, or no value if slowstart is inactive.

type: scaled_float

format: percent


**`haproxy.stat.selected.total`**
:   Total number of times a server was selected, either for new sessions, or when re-dispatching. For servers, this field reports the the number of times the server was selected.

type: long


**`haproxy.stat.tracked.id`**
:   ID of the proxy/server if tracking is enabled.

type: long


**`haproxy.stat.cookie`**
:   Cookie value of the server or the name of the cookie of the backend.

type: keyword


**`haproxy.stat.load_balancing_algorithm`**
:   Load balancing algorithm.

type: keyword


**`haproxy.stat.connection.total`**
:   Cumulative number of connections.

type: long


**`haproxy.stat.connection.retried`**
:   Number of times a connection to a server was retried.

type: long


**`haproxy.stat.connection.time.avg`**
:   Average connect time in ms over the last 1024 requests.

type: long


**`haproxy.stat.connection.rate`**
:   Number of connections over the last second.

type: long


**`haproxy.stat.connection.rate_max`**
:   Highest value of connection.rate.

type: long


**`haproxy.stat.connection.attempt.total`**
:   Number of connection establishment attempts.

type: long


**`haproxy.stat.connection.reuse.total`**
:   Number of connection reuses.

type: long


**`haproxy.stat.connection.idle.total`**
:   Number of idle connections available for reuse.

type: long


**`haproxy.stat.connection.idle.limit`**
:   Limit on idle connections available for reuse.

type: long


**`haproxy.stat.connection.cache.lookup.total`**
:   Number of cache lookups.

type: long


**`haproxy.stat.connection.cache.hits`**
:   Number of cache hits.

type: long


**`haproxy.stat.request.denied`**
:   Requests denied because of security concerns.

* For TCP this is because of a matched tcp-request content rule.
* For HTTP this is because of a matched http-request or tarpit rule.

type: long


**`haproxy.stat.request.denied_by_connection_rules`**
:   Requests denied because of TCP request connection rules.

type: long


**`haproxy.stat.request.denied_by_session_rules`**
:   Requests denied because of TCP request session rules.

type: long


**`haproxy.stat.request.queued.current`**
:   Current queued requests. For backends, this field reports the number of requests queued without a server assigned.

type: long


**`haproxy.stat.request.queued.max`**
:   Maximum value of queued.current.

type: long


**`haproxy.stat.request.errors`**
:   Request errors. Some of the possible causes are:

* early termination from the client, before the request has been sent
* read error from the client
* client timeout
* client closed connection
* various bad requests from the client.
* request was tarpitted.

type: long


**`haproxy.stat.request.redispatched`**
:   Number of times a request was redispatched to another server. For servers, this field reports the number of times the server was switched away from.

type: long


**`haproxy.stat.request.connection.errors`**
:   Number of requests that encountered an error trying to connect to a server. For backends, this field reports the sum of the stat for all backend servers, plus any connection errors not associated with a particular server (such as the backend having no active servers).

type: long



## rate [_rate_2]

**`haproxy.stat.request.rate.value`**
:   Number of HTTP requests per second over the last elapsed second.

type: long


**`haproxy.stat.request.rate.max`**
:   Maximum number of HTTP requests per second.

type: long


**`haproxy.stat.request.total`**
:   Total number of HTTP requests received.

type: long


**`haproxy.stat.request.intercepted`**
:   Number of intercepted requests.

type: long


**`haproxy.stat.response.errors`**
:   Number of response errors. This value includes the number of data transfers aborted by the server (haproxy.stat.server.aborted). Some other errors are: * write errors on the client socket (won’t be counted for the server stat) * failure applying filters to the response

type: long


**`haproxy.stat.response.time.avg`**
:   Average response time in ms over the last 1024 requests (0 for TCP).

type: long


**`haproxy.stat.response.denied`**
:   Responses denied because of security concerns. For HTTP this is because of a matched http-request rule, or "option checkcache".

type: integer



## http [_http_3]

**`haproxy.stat.response.http.1xx`**
:   HTTP responses with 1xx code.

type: long


**`haproxy.stat.response.http.2xx`**
:   HTTP responses with 2xx code.

type: long


**`haproxy.stat.response.http.3xx`**
:   HTTP responses with 3xx code.

type: long


**`haproxy.stat.response.http.4xx`**
:   HTTP responses with 4xx code.

type: long


**`haproxy.stat.response.http.5xx`**
:   HTTP responses with 5xx code.

type: long


**`haproxy.stat.response.http.other`**
:   HTTP responses with other codes (protocol error).

type: long


**`haproxy.stat.header.rewrite.failed.total`**
:   Number of failed header rewrite warnings.

type: long


**`haproxy.stat.session.current`**
:   Number of current sessions.

type: long


**`haproxy.stat.session.max`**
:   Maximum number of sessions.

type: long


**`haproxy.stat.session.limit`**
:   Configured session limit.

type: long


**`haproxy.stat.session.total`**
:   Number of all sessions.

type: long


**`haproxy.stat.session.rate.value`**
:   Number of sessions per second over the last elapsed second.

type: integer


**`haproxy.stat.session.rate.limit`**
:   Configured limit on new sessions per second.

type: integer


**`haproxy.stat.session.rate.max`**
:   Maximum number of new sessions per second.

type: integer



## check [_check]

**`haproxy.stat.check.status`**
:   Status of the last health check. One of:

```
UNK     -> unknown
INI     -> initializing
SOCKERR -> socket error
L4OK    -> check passed on layer 4, no upper layers testing enabled
L4TOUT  -> layer 1-4 timeout
L4CON   -> layer 1-4 connection problem, for example
          "Connection refused" (tcp rst) or "No route to host" (icmp)
L6OK    -> check passed on layer 6
L6TOUT  -> layer 6 (SSL) timeout
L6RSP   -> layer 6 invalid response - protocol error
L7OK    -> check passed on layer 7
L7OKC   -> check conditionally passed on layer 7, for example 404 with
          disable-on-404
L7TOUT  -> layer 7 (HTTP/SMTP) timeout
L7RSP   -> layer 7 invalid response - protocol error
L7STS   -> layer 7 response error, for example HTTP 5xx
```
type: keyword


**`haproxy.stat.check.code`**
:   Layer 5-7 code, if available.

type: long


**`haproxy.stat.check.duration`**
:   Time in ms that it took to finish the last health check.

type: long


**`haproxy.stat.check.health.last`**
:   The result of the last health check.

type: keyword


**`haproxy.stat.check.health.fail`**
:   Number of failed checks.

type: long


**`haproxy.stat.check.agent.last`**
:   type: integer


**`haproxy.stat.check.failed`**
:   Number of checks that failed while the server was up.

type: long


**`haproxy.stat.check.down`**
:   Number of UP→DOWN transitions. For backends, this value is the number of transitions to the whole backend being down, rather than the sum of the transitions for each server.

type: long


**`haproxy.stat.client.aborted`**
:   Number of data transfers aborted by the client.

type: integer



## server [_server_7]

**`haproxy.stat.server.id`**
:   Server ID (unique inside a proxy).

type: integer


**`haproxy.stat.server.aborted`**
:   Number of data transfers aborted by the server. This value is included in haproxy.stat.response.errors.

type: integer


**`haproxy.stat.server.active`**
:   Number of backend servers that are active, meaning that they are healthy and can receive requests from the load balancer.

type: integer


**`haproxy.stat.server.backup`**
:   Number of backend servers that are backup servers.

type: integer



## compressor [_compressor]

**`haproxy.stat.compressor.in.bytes`**
:   Number of HTTP response bytes fed to the compressor.

type: long

format: bytes


**`haproxy.stat.compressor.out.bytes`**
:   Number of HTTP response bytes emitted by the compressor.

type: integer

format: bytes


**`haproxy.stat.compressor.bypassed.bytes`**
:   Number of bytes that bypassed the HTTP compressor (CPU/BW limit).

type: long

format: bytes


**`haproxy.stat.compressor.response.bytes`**
:   Number of HTTP responses that were compressed.

type: long

format: bytes



## proxy [_proxy_2]

**`haproxy.stat.proxy.id`**
:   Unique proxy ID.

type: integer


**`haproxy.stat.proxy.name`**
:   Proxy name.

type: keyword


**`haproxy.stat.proxy.mode`**
:   Proxy mode (tcp, http, health, unknown).

type: keyword



## queue [_queue_8]

**`haproxy.stat.queue.limit`**
:   Configured queue limit (maxqueue) for the server, or nothing if the value of maxqueue is 0 (meaning no limit).

type: integer


**`haproxy.stat.queue.time.avg`**
:   The average queue time in ms over the last 1024 requests.

type: integer



## agent [_agent_3]

**`haproxy.stat.agent.status`**
:   Status of the last health check. One of:

```
UNK     -> unknown
INI     -> initializing
SOCKERR -> socket error
L4OK    -> check passed on layer 4, no upper layers enabled
L4TOUT  -> layer 1-4 timeout
L4CON   -> layer 1-4 connection problem, for example
          "Connection refused" (tcp rst) or "No route to host" (icmp)
L7OK    -> agent reported "up"
L7STS   -> agent reported "fail", "stop" or "down"
```
type: keyword


**`haproxy.stat.agent.description`**
:   Human readable version of agent.status.

type: keyword


**`haproxy.stat.agent.code`**
:   Value reported by agent.

type: integer


**`haproxy.stat.agent.rise`**
:   Rise value of agent.

type: integer


**`haproxy.stat.agent.fall`**
:   Fall value of agent.

type: integer


**`haproxy.stat.agent.health`**
:   Health parameter of agent. Between 0 and `agent.rise`+`agent.fall`-1.

type: integer


**`haproxy.stat.agent.duration`**
:   Duration of the last check in ms.

type: integer


**`haproxy.stat.agent.check.rise`**
:   Rise value of server.

type: integer


**`haproxy.stat.agent.check.fall`**
:   Fall value of server.

type: integer


**`haproxy.stat.agent.check.health`**
:   Health parameter of server. Between 0 and `agent.check.rise`+`agent.check.fall`-1.

type: integer


**`haproxy.stat.agent.check.description`**
:   Human readable version of check.

type: keyword


**`haproxy.stat.source.address`**
:   Address of the source.

type: text


