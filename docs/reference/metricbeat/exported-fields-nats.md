---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-nats.html
---

# NATS fields [exported-fields-nats]

nats Module


## nats [_nats]

`nats` contains statistics that were read from Nats

**`nats.server.id`**
:   The server ID

type: keyword


**`nats.server.time`**
:   [8.0.0]

Server time of metric creation

type: date



## connection [_connection_3]

Contains nats connection related metrics

**`nats.connection.name`**
:   The name of the connection

type: keyword


**`nats.connection.subscriptions`**
:   The number of subscriptions in this connection

type: integer


**`nats.connection.pending_bytes`**
:   The number of pending bytes of this connection

type: long

format: bytes


**`nats.connection.uptime`**
:   The period the connection is up (sec)

type: long

format: duration


**`nats.connection.idle_time`**
:   The period the connection is idle (sec)

type: long

format: duration



## in [_in_2]

The amount of incoming data

**`nats.connection.in.messages`**
:   The amount of incoming messages

type: long


**`nats.connection.in.bytes`**
:   The amount of incoming bytes

type: long

format: bytes



## out [_out_2]

The amount of outgoing data

**`nats.connection.out.messages`**
:   The amount of outgoing messages

type: long


**`nats.connection.out.bytes`**
:   The amount of outgoing bytes

type: long

format: bytes



## connections [_connections_4]

Contains nats connection related metrics

**`nats.connections.total`**
:   The number of currently active clients

type: integer



## route [_route]

Contains nats route related metrics

**`nats.route.subscriptions`**
:   The number of subscriptions in this connection

type: integer


**`nats.route.remote_id`**
:   The remote id on which the route is connected to

type: keyword


**`nats.route.pending_size`**
:   The number of pending routes

type: long


**`nats.route.port`**
:   The port of the route

type: integer


**`nats.route.ip`**
:   The ip of the route

type: ip



## in [_in_3]

The amount of incoming data

**`nats.route.in.messages`**
:   The amount of incoming messages

type: long


**`nats.route.in.bytes`**
:   The amount of incoming bytes

type: long

format: bytes



## out [_out_3]

The amount of outgoing data

**`nats.route.out.messages`**
:   The amount of outgoing messages

type: long


**`nats.route.out.bytes`**
:   The amount of outgoing bytes

type: long

format: bytes



## routes [_routes]

Contains nats route related metrics

**`nats.routes.total`**
:   The number of registered routes

type: integer



## stats [_stats_9]

Contains nats var related metrics

**`nats.stats.uptime`**
:   The period the server is up (sec)

type: long

format: duration


**`nats.stats.mem.bytes`**
:   The current memory usage of NATS process

type: long

format: bytes


**`nats.stats.cores`**
:   The number of logical cores the NATS process runs on

type: integer


**`nats.stats.cpu`**
:   The current cpu usage of NATs process

type: scaled_float

format: percent


**`nats.stats.total_connections`**
:   The number of totally created clients

type: long


**`nats.stats.remotes`**
:   The number of registered remotes

type: integer



## in [_in_4]

The amount of incoming data

**`nats.stats.in.messages`**
:   The amount of incoming messages

type: long


**`nats.stats.in.bytes`**
:   The amount of incoming bytes

type: long

format: bytes



## out [_out_4]

The amount of outgoing data

**`nats.stats.out.messages`**
:   The amount of outgoing messages

type: long


**`nats.stats.out.bytes`**
:   The amount of outgoing bytes

type: long

format: bytes


**`nats.stats.slow_consumers`**
:   The number of slow consumers currently on NATS

type: long



## http [_http_5]

The http metrics of NATS server


## req_stats [_req_stats]

The requests statistics


## uri [_uri]

The request distribution on monitoring URIS

**`nats.stats.http.req_stats.uri.routez`**
:   The number of hits on routez monitoring uri

type: long


**`nats.stats.http.req_stats.uri.connz`**
:   The number of hits on connz monitoring uri

type: long


**`nats.stats.http.req_stats.uri.varz`**
:   The number of hits on varz monitoring uri

type: long


**`nats.stats.http.req_stats.uri.subsz`**
:   The number of hits on subsz monitoring uri

type: long


**`nats.stats.http.req_stats.uri.root`**
:   The number of hits on root monitoring uri

type: long



## subscriptions [_subscriptions]

Contains nats subscriptions related metrics

**`nats.subscriptions.total`**
:   The number of active subscriptions

type: integer


**`nats.subscriptions.inserts`**
:   The number of insert operations in subscriptions list

type: long


**`nats.subscriptions.removes`**
:   The number of remove operations in subscriptions list

type: long


**`nats.subscriptions.matches`**
:   The number of times a match is found for a subscription

type: long


**`nats.subscriptions.cache.size`**
:   The number of result sets in the cache

type: integer


**`nats.subscriptions.cache.hit_rate`**
:   The rate matches are being retrieved from cache

type: scaled_float

format: percent


**`nats.subscriptions.cache.fanout.max`**
:   The maximum fanout served by cache

type: integer


**`nats.subscriptions.cache.fanout.avg`**
:   The average fanout served by cache

type: double


