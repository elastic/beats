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
:   :::{admonition} Deprecated in 8.0.0
    The `nats.server.time` field was deprecated in 8.0.0.
    :::

Server time of metric creation

type: date



## connection [_connection_3]

Contains nats connection related metrics

**`nats.connection.id`**
:   The ID of the connection

type: keyword

**`nats.connection.name`**
:   The name of the connection

type: keyword

**`nats.connection.kind`**
:   The kind of the connection

type: keyword

**`nats.connection.type`**
:   The type of the connection

type: keyword

**`nats.connection.ip`**
:   The IP address of the connection

type: ip

**`nats.connection.port`**
:   The port of the connection

type: integer

**`nats.connection.lang`**
:   The language of the client connection

type: keyword

**`nats.connection.version`**
:   The version of the client connection

type: keyword

**`nats.connection.start`**
:   The time the connection was started

type: date

**`nats.connection.last_activity`**
:   The last activity time of the connection

type: date

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


### in [_in_2]

The amount of incoming data

**`nats.connection.in.messages`**
:   The amount of incoming messages

type: long


**`nats.connection.in.bytes`**
:   The amount of incoming bytes

type: long

format: bytes


### out [_out_2]

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
:   The remote ID on which the route is connected

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


### in [_in_3]

The amount of incoming data

**`nats.route.in.messages`**
:   The amount of incoming messages

type: long


**`nats.route.in.bytes`**
:   The amount of incoming bytes

type: long

format: bytes


### out [_out_3]

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

**`nats.stats.server_name`**
:   The name of the NATS server

type: keyword

**`nats.stats.version`**
:   The version of the NATS server

type: keyword

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

**`nats.stats.slow_consumers`**
:   The number of slow consumers currently on NATS

type: long


### in [_in_4]

The amount of incoming data

**`nats.stats.in.messages`**
:   The amount of incoming messages

type: long


**`nats.stats.in.bytes`**
:   The amount of incoming bytes

type: long

format: bytes


### out [_out_4]

The amount of outgoing data

**`nats.stats.out.messages`**
:   The amount of outgoing messages

type: long

**`nats.stats.out.bytes`**
:   The amount of outgoing bytes

type: long

format: bytes


### http [_http_5]

The http metrics of NATS server

#### req_stats [_req_stats]

The requests statistics

##### uri [_uri]

The request distribution on monitoring URIs

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

**`nats.stats.http.req_stats.uri.jsz`**
:   The number of hits on jsz monitoring uri

type: long

**`nats.stats.http.req_stats.uri.accountz`**
:   The number of hits on accountz monitoring uri

type: long

**`nats.stats.http.req_stats.uri.accstatz`**
:   The number of hits on accstatz monitoring uri

type: long

**`nats.stats.http.req_stats.uri.gatewayz`**
:   The number of hits on gatewayz monitoring uri

type: long

**`nats.stats.http.req_stats.uri.healthz`**
:   The number of hits on healthz monitoring uri

type: long

**`nats.stats.http.req_stats.uri.leafz`**
:   The number of hits on leafz monitoring uri

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

## jetstream [_jetstream]

Information pertaining to a NATS JetStream server

**`nats.jetstream.category`**
:   The category of metrics represented in this event (stats, account, stream, or consumer).

type: keyword

### stats [_jetstream_stats]

General stats about the NATS JetStream server.

**`nats.jetstream.stats.streams`**
:   The total number of streams on the JetStream server.

type: long

**`nats.jetstream.stats.consumers`**
:   The total number of consumers on the JetStream server.

type: long

**`nats.jetstream.stats.messages`**
:   The total number of messages on the JetStream server.

type: long

**`nats.jetstream.stats.bytes`**
:   The total number of message bytes on the JetStream server.

type: long

format: bytes

**`nats.jetstream.stats.memory`**
:   The total amount of memory (bytes) used by the JetStream server.

type: long

format: bytes

**`nats.jetstream.stats.reserved_memory`**
:   The of memory (bytes) reserved by the JetStream server.

type: long

format: bytes

**`nats.jetstream.stats.storage`**
:   The total amount of storage (bytes) used by the JetStream server.

type: long

format: bytes

**`nats.jetstream.stats.reserved_storage`**
:   The total amount of storage (bytes) reserved by the JetStream server.

type: long

format: bytes

**`nats.jetstream.stats.accounts`**
:   The total number of accounts on the JetStream server.

type: long

#### config [_jetstream_stats_config]

Configuration of the JetStream server.

**`nats.jetstream.stats.config.max_memory`**
:   The maximum amount of memory (bytes) the JetStream server can use.

type: long

format: bytes

**`nats.jetstream.stats.config.max_storage`**
:   The maximum amount of storage (bytes) the JetStream server can use.

type: long

format: bytes

**`nats.jetstream.stats.config.store_dir`**
:   The path on disk where the JetStream storage lives.

type: keyword

**`nats.jetstream.stats.config.sync_interval`**
:   The fsync/sync interval for page cache in the filestore.

type: long

### account [_jetstream_account]

Information about a NATS JetStream account.

**`nats.jetstream.account.id`**
:   The ID of the JetStream account.

type: keyword

**`nats.jetstream.account.name`**
:   The name of the JetStream account.

type: keyword

**`nats.jetstream.account.accounts`**
:   The number of accounts using JetStream on the server.

type: integer

**`nats.jetstream.account.high_availability_assets`**
:   Indicates the number of JetStream high-availability (HA) assets allocated for an account.

type: integer

**`nats.jetstream.account.memory`**
:   The amount of memory in bytes currently used by JetStream for this account.

type: long

format: bytes

**`nats.jetstream.account.storage`**
:   The amount of storage in bytes currently used by JetStream for this account.

type: long

format: bytes

**`nats.jetstream.account.reserved_memory`**
:   The maximum memory quota reserved for this account (in bytes).

type: long

format: bytes

**`nats.jetstream.account.reserved_storage`**
:   The maximum disk storage quota reserved for this account (in bytes).

type: long

format: bytes

#### api [_jetstream_account_api]

API stats pertaining to this account.

**`nats.jetstream.account.api.total`**
:   The total number of JetStream API calls made by this account.

type: long

**`nats.jetstream.account.api.errors`**
:   The total number of JetStream API errors encountered by this account.

type: long

### stream [_jetstream_stream]

Information about a NATS JetStream stream.

**`nats.jetstream.stream.name`**
:   The name of the JetStream stream.

type: keyword

**`nats.jetstream.stream.created`**
:   The date/time the stream was created.

type: date

#### cluster [_jetstream_stream_cluster]

Cluster information for the stream.

**`nats.jetstream.stream.cluster.leader`**
:   The ID of the leader in the cluster.

type: keyword

#### state [_jetstream_stream_state]

The state of the stream.

**`nats.jetstream.stream.state.messages`**
:   The number of messages on the stream.

type: long

**`nats.jetstream.stream.state.bytes`**
:   The number of bytes of messages on the stream.

type: long

format: bytes

**`nats.jetstream.stream.state.consumer_count`**
:   The number of consumers on the stream.

type: long

**`nats.jetstream.stream.state.num_subjects`**
:   The number of subjects on the stream.

type: long

**`nats.jetstream.stream.state.num_deleted`**
:   The number of messages deleted from the stream.

type: long

**`nats.jetstream.stream.state.first_seq`**
:   The first sequence number on the stream.

type: long

**`nats.jetstream.stream.state.first_ts`**
:   The date/time corresponding to first_seq.

type: date

**`nats.jetstream.stream.state.last_seq`**
:   The last sequence number on the stream.

type: long

**`nats.jetstream.stream.state.last_ts`**
:   The date/time corresponding to last_seq.

type: date

#### account [_jetstream_stream_account]

Information about the account for this stream.

**`nats.jetstream.stream.account.id`**
:   The ID of the account.


type: keyword

**`nats.jetstream.stream.account.name`**
:   The name of the account.

type: keyword

#### config [_jetstream_stream_config]

Information regarding how the stream is configured.

**`nats.jetstream.stream.config.description`**
:   The description of the stream.

type: text

**`nats.jetstream.stream.config.retention`**
:   The retention policy for the stream.

type: keyword

**`nats.jetstream.stream.config.num_replicas`**
:   How many replicas to keep for each message in a clustered JetStream.

type: integer

**`nats.jetstream.stream.config.storage`**
:   The storage type for stream data.

type: keyword

**`nats.jetstream.stream.config.max_consumers`**
:   The maximum number of consumers allowed for this stream.

type: long

**`nats.jetstream.stream.config.max_msgs`**
:   Maximum number of messages stored in the stream. Adheres to Discard Policy, removing oldest or refusing new messages if the Stream exceeds this number of messages.

type: long

**`nats.jetstream.stream.config.max_bytes`**
:   Maximum number of bytes stored in the stream. Adheres to Discard Policy, removing oldest or refusing new messages if the Stream exceeds this size.

type: long

format: bytes

**`nats.jetstream.stream.config.max_age`**
:   Maximum age of any message in the stream, expressed in nanoseconds.	

type: long

**`nats.jetstream.stream.config.max_msgs_per_subject`**
:   Limits maximum number of messages in the stream to retain per subject.	

type: long

**`nats.jetstream.stream.config.max_msg_size`**
:   The largest message (bytes) that will be accepted by the stream. The size of a message is a sum of payload and headers.

type: long

format: bytes

**`nats.jetstream.stream.config.subjects`**
:   The list of subjects bound to the stream.

type: keyword

### consumer [_jetstream_consumer]

Information about a NATS JetStream consumer.

**`nats.jetstream.consumer.name`**
:   The name of the consumer.

type: keyword

**`nats.jetstream.consumer.created`**
:   The date/time the consumer was created.

type: date

#### stream [_jetstream_consumer_stream]

Information about the stream for this consumer.

**`nats.jetstream.consumer.stream.name`**
:   The name of the stream.

type: keyword

#### cluster [_jetstream_consumer_cluster]

Cluster information for the consumer.

**`nats.jetstream.consumer.cluster.leader`**
:   The ID of the leader in the cluster.

type: keyword

#### ack_floor [_jetstream_consumer_ack_floor]

Information about message acknowledgements pertaining to AckFloor, which indicates the highest contiguous sequence number that has been fully acknowledged.

**`nats.jetstream.consumer.ack_floor.consumer_seq`**
:   The lowest contiguous consumer sequence number that has been acknowledged.

type: long

**`nats.jetstream.consumer.ack_floor.stream_seq`**
:   The lowest contiguous stream sequence number that has been acknowledged by the consumer.

type: long

**`nats.jetstream.consumer.ack_floor.last_active`**
:   The timestamp of the last acknowledged message.

type: date

#### delivered [_jetstream_consumer_delivered]

Information about delivered messages.

**`nats.jetstream.consumer.delivered.consumer_seq`**
:   The number of messages delivered to this consumer, starting from 1 when the consumer was created.

type: long

**`nats.jetstream.consumer.delivered.stream_seq`**
:   The last stream sequence number of a message delivered to the consumer. Corresponds to the global sequence of messages in the stream.

type: long

**`nats.jetstream.consumer.delivered.last_active`**
:   The timestamp of the last message delivered to the consumer.

type: date

**`nats.jetstream.consumer.num_ack_pending`**
:   The number of messages that have been delivered to the consumer but not yet acknowledged.

type: long

**`nats.jetstream.consumer.num_redelivered`**
:   The number of messages that had to be resent because they were previously delivered but not acknowledged within the Ack Wait time.

type: long

**`nats.jetstream.consumer.num_waiting`**
:   The number of pull requests currently waiting for messages to be delivered.

type: long

**`nats.jetstream.consumer.num_pending`**
:   The number of messages remaining in the stream that the consumer has not yet delivered to any client.

type: long

**`nats.jetstream.consumer.last_active_time`**
:   Represents the last activity time of the consumer.

type: date

#### account [_jetstream_consumer_account]

Information about the account for this consumer.

**`nats.jetstream.consumer.account.id`**
:   The ID of the account.

type: keyword

**`nats.jetstream.consumer.account.name`**
:   The name of the account.

type: keyword

#### config [_jetstream_consumer_config]

Information about the configuration for this consumer.

**`nats.jetstream.consumer.config.name`**
:   The name of the consumer.

type: keyword

**`nats.jetstream.consumer.config.durable_name`**
:   The durable name of the consumer. If set, clients can have subscriptions bind to the consumer and resume until the consumer is explicitly deleted.

type: keyword

**`nats.jetstream.consumer.config.deliver_policy`**
:   The point in the stream from which to receive messages.

type: keyword

**`nats.jetstream.consumer.config.filter_subject`**
:   A subject that overlaps with the subjects bound to the stream to filter delivery to subscribers.

type: keyword

**`nats.jetstream.consumer.config.replay_policy`**
:   The configured replay policy for the consumer.

type: keyword

**`nats.jetstream.consumer.config.ack_policy`**
:   The configured ack policy for the consumer.

type: keyword

**`nats.jetstream.consumer.config.ack_wait`**
:   The duration (in nanoseconds) that the server will wait for an acknowledgment for any individual message once it has been delivered to a consumer. If an acknowledgment is not received in time, the message will be redelivered.

type: long

**`nats.jetstream.consumer.config.max_deliver`**
:   The maximum number of times a message will be redelivered if not acknowledged.

type: long

**`nats.jetstream.consumer.config.max_waiting`**
:   The maximum number of pull requests a consumer can have waiting for messages.

type: long

**`nats.jetstream.consumer.config.max_ack_pending`**
:   The maximum number of messages the consumer can have in-flight (delivered but unacknowledged) at any time.

type: long

**`nats.jetstream.consumer.config.num_replicas`**
:   The number of replicas for the consumer's state in a JetStream cluster.

type: long
