---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-php_fpm.html
---

# PHP_FPM fields [exported-fields-php_fpm]

PHP-FPM server status metrics collected from PHP-FPM.


## php_fpm [_php_fpm]

`php_fpm` contains the metrics that were obtained from PHP-FPM status page call.


## pool [_pool_3]

`pool` contains the metrics that were obtained from the PHP-FPM process pool.

**`php_fpm.pool.name`**
:   The name of the pool.

type: keyword



## pool [_pool_4]

`pool` contains the metrics that were obtained from the PHP-FPM process pool.

**`php_fpm.pool.process_manager`**
:   Static, dynamic or ondemand.

type: keyword



## connections [_connections_5]

Connection state specific statistics.

**`php_fpm.pool.connections.accepted`**
:   The number of incoming requests that the PHP-FPM server has accepted; when a connection is accepted it is removed from the listen queue.

type: long


**`php_fpm.pool.connections.queued`**
:   The current number of connections that have been initiated, but not yet accepted. If this value is non-zero it typically means that all the available server processes are currently busy, and there are no processes available to serve the next request. Raising `pm.max_children` (provided the server can handle it) should help keep this number low. This property follows from the fact that PHP-FPM listens via a socket (TCP or file based), and thus inherits some of the characteristics of sockets.

type: long


**`php_fpm.pool.connections.max_listen_queue`**
:   The maximum number of requests in the queue of pending connections since FPM has started.

type: long


**`php_fpm.pool.connections.listen_queue_len`**
:   The size of the socket queue of pending connections.

type: long



## processes [_processes]

Process state specific statistics.

**`php_fpm.pool.processes.idle`**
:   The number of servers in the `waiting to process` state (i.e. not currently serving a page). This value should fall between the `pm.min_spare_servers` and `pm.max_spare_servers` values when the process manager is `dynamic`.

type: long


**`php_fpm.pool.processes.active`**
:   The number of servers current processing a page - the minimum is `1` (so even on a fully idle server, the result will be not read `0`).

type: long


**`php_fpm.pool.processes.total`**
:   The number of idle + active processes.

type: long


**`php_fpm.pool.processes.max_active`**
:   The maximum number of active processes since FPM has started.

type: long


**`php_fpm.pool.processes.max_children_reached`**
:   Number of times, the process limit has been reached, when pm tries to start more children (works only for pm *dynamic* and *ondemand*).

type: long


**`php_fpm.pool.slow_requests`**
:   The number of times a request execution time has exceeded `request_slowlog_timeout`.

type: long


**`php_fpm.pool.start_since`**
:   Number of seconds since FPM has started.

type: long


**`php_fpm.pool.start_time`**
:   The date and time FPM has started.

type: date



## process [_process_7]

process contains the metrics that were obtained from the PHP-FPM process.

**`php_fpm.process.pid`**
:   The PID of the process

type: alias

alias to: process.pid


**`php_fpm.process.state`**
:   The state of the process (Idle, Running, etc)

type: keyword


**`php_fpm.process.start_time`**
:   The date and time the process has started

type: date


**`php_fpm.process.start_since`**
:   The number of seconds since the process has started

type: integer


**`php_fpm.process.requests`**
:   The number of requests the process has served

type: integer


**`php_fpm.process.request_duration`**
:   The duration in microseconds (1 million in a second) of the current request (my own definition)

type: integer


**`php_fpm.process.request_method`**
:   The request method (GET, POST, etc) (of the current request)

type: alias

alias to: http.request.method


**`php_fpm.process.request_uri`**
:   The request URI with the query string (of the current request)

type: alias

alias to: url.original


**`php_fpm.process.content_length`**
:   The content length of the request (only with POST) (of the current request)

type: alias

alias to: http.response.body.bytes


**`php_fpm.process.user`**
:   The user (PHP_AUTH_USER) (or - if not set) (for the current request)

type: alias

alias to: user.name


**`php_fpm.process.script`**
:   The main script called (or - if not set) (for the current request)

type: keyword


**`php_fpm.process.last_request_cpu`**
:   The CPU percentage the last request consumed. It’s always 0 if the process is not in Idle state because CPU calculation is done when the request processing has terminated

type: long


**`php_fpm.process.last_request_memory`**
:   The max amount of memory the last request consumed. It’s always 0 if the process is not in Idle state because memory calculation is done when the request processing has terminated

type: integer


