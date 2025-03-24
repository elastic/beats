---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-rabbitmq.html
---

# RabbitMQ fields [exported-fields-rabbitmq]

RabbitMQ module


## rabbitmq [_rabbitmq]

**`rabbitmq.vhost`**
:   Virtual host name with non-ASCII characters escaped as in C.

type: keyword



## connection [_connection_4]

connection

**`rabbitmq.connection.name`**
:   The name of the connection with non-ASCII characters escaped as in C.

type: keyword


**`rabbitmq.connection.vhost`**
:   Virtual host name with non-ASCII characters escaped as in C.

type: alias

alias to: rabbitmq.vhost


**`rabbitmq.connection.user`**
:   User name.

type: alias

alias to: user.name


**`rabbitmq.connection.node`**
:   Node name.

type: alias

alias to: rabbitmq.node.name


**`rabbitmq.connection.state`**
:   Connection state.

type: keyword


**`rabbitmq.connection.channels`**
:   The number of channels on the connection.

type: long


**`rabbitmq.connection.channel_max`**
:   The maximum number of channels allowed on the connection.

type: long


**`rabbitmq.connection.frame_max`**
:   Maximum permissible size of a frame (in bytes) to negotiate with clients.

type: long

format: bytes


**`rabbitmq.connection.type`**
:   Type of the connection.

type: keyword


**`rabbitmq.connection.host`**
:   Server hostname obtained via reverse DNS, or its IP address if reverse DNS failed or was disabled.

type: keyword


**`rabbitmq.connection.peer.host`**
:   Peer hostname obtained via reverse DNS, or its IP address if reverse DNS failed or was not enabled.

type: keyword


**`rabbitmq.connection.port`**
:   Server port.

type: long


**`rabbitmq.connection.peer.port`**
:   Peer port.

type: long


**`rabbitmq.connection.packet_count.sent`**
:   Number of packets sent on the connection.

type: long


**`rabbitmq.connection.packet_count.received`**
:   Number of packets received on the connection.

type: long


**`rabbitmq.connection.packet_count.pending`**
:   Number of packets pending on the connection.

type: long


**`rabbitmq.connection.octet_count.sent`**
:   Number of octets sent on the connection.

type: long


**`rabbitmq.connection.octet_count.received`**
:   Number of octets received on the connection.

type: long


**`rabbitmq.connection.client_provided.name`**
:   User specified connection name.

type: keyword



## exchange [_exchange]

exchange

**`rabbitmq.exchange.name`**
:   The name of the queue with non-ASCII characters escaped as in C.

type: keyword


**`rabbitmq.exchange.vhost`**
:   Virtual host name with non-ASCII characters escaped as in C.

type: alias

alias to: rabbitmq.vhost


**`rabbitmq.exchange.durable`**
:   Whether or not the queue survives server restarts.

type: boolean


**`rabbitmq.exchange.auto_delete`**
:   Whether the queue will be deleted automatically when no longer used.

type: boolean


**`rabbitmq.exchange.internal`**
:   Whether the exchange is internal, i.e. cannot be directly published to by a client.

type: boolean


**`rabbitmq.exchange.user`**
:   User who created the exchange.

type: alias

alias to: user.name


**`rabbitmq.exchange.messages.publish_in.count`**
:   Count of messages published "in" to an exchange, i.e. not taking account of routing.

type: long


**`rabbitmq.exchange.messages.publish_in.details.rate`**
:   How much the exchange publish-in count has changed per second in the most recent sampling interval.

type: float


**`rabbitmq.exchange.messages.publish_out.count`**
:   Count of messages published "out" of an exchange, i.e. taking account of routing.

type: long


**`rabbitmq.exchange.messages.publish_out.details.rate`**
:   How much the exchange publish-out count has changed per second in the most recent sampling interval.

type: float



## node [_node_8]

node

**`rabbitmq.node.disk.free.bytes`**
:   Disk free space in bytes.

type: long

format: bytes


**`rabbitmq.node.disk.free.limit.bytes`**
:   Point at which the disk alarm will go off.

type: long

format: bytes


**`rabbitmq.node.fd.total`**
:   File descriptors available.

type: long


**`rabbitmq.node.fd.used`**
:   Used file descriptors.

type: long


**`rabbitmq.node.gc.num.count`**
:   Number of GC operations.

type: long


**`rabbitmq.node.gc.reclaimed.bytes`**
:   GC bytes reclaimed.

type: long

format: bytes


**`rabbitmq.node.io.file_handle.open_attempt.avg.ms`**
:   File handle open avg time

type: long


**`rabbitmq.node.io.file_handle.open_attempt.count`**
:   File handle open attempts

type: long


**`rabbitmq.node.io.read.avg.ms`**
:   File handle read avg time

type: long


**`rabbitmq.node.io.read.bytes`**
:   Data read in bytes

type: long

format: bytes


**`rabbitmq.node.io.read.count`**
:   Data read operations

type: long


**`rabbitmq.node.io.reopen.count`**
:   Data reopen operations

type: long


**`rabbitmq.node.io.seek.avg.ms`**
:   Data seek avg time

type: long


**`rabbitmq.node.io.seek.count`**
:   Data seek operations

type: long


**`rabbitmq.node.io.sync.avg.ms`**
:   Data sync avg time

type: long


**`rabbitmq.node.io.sync.count`**
:   Data sync operations

type: long


**`rabbitmq.node.io.write.avg.ms`**
:   Data write avg time

type: long


**`rabbitmq.node.io.write.bytes`**
:   Data write in bytes

type: long

format: bytes


**`rabbitmq.node.io.write.count`**
:   Data write operations

type: long


**`rabbitmq.node.mem.limit.bytes`**
:   Point at which the memory alarm will go off.

type: long

format: bytes


**`rabbitmq.node.mem.used.bytes`**
:   Memory used in bytes.

type: long


**`rabbitmq.node.mnesia.disk.tx.count`**
:   Number of Mnesia transactions which have been performed that required writes to disk.

type: long


**`rabbitmq.node.mnesia.ram.tx.count`**
:   Number of Mnesia transactions which have been performed that did not require writes to disk.

type: long


**`rabbitmq.node.msg.store_read.count`**
:   Number of messages which have been read from the message store.

type: long


**`rabbitmq.node.msg.store_write.count`**
:   Number of messages which have been written to the message store.

type: long


**`rabbitmq.node.name`**
:   Node name

type: keyword


**`rabbitmq.node.proc.total`**
:   Maximum number of Erlang processes.

type: long


**`rabbitmq.node.proc.used`**
:   Number of Erlang processes in use.

type: long


**`rabbitmq.node.processors`**
:   Number of cores detected and usable by Erlang.

type: long


**`rabbitmq.node.queue.index.journal_write.count`**
:   Number of records written to the queue index journal.

type: long


**`rabbitmq.node.queue.index.read.count`**
:   Number of records read from the queue index.

type: long


**`rabbitmq.node.queue.index.write.count`**
:   Number of records written to the queue index.

type: long


**`rabbitmq.node.run.queue`**
:   Average number of Erlang processes waiting to run.

type: long


**`rabbitmq.node.socket.total`**
:   File descriptors available for use as sockets.

type: long


**`rabbitmq.node.socket.used`**
:   File descriptors used as sockets.

type: long


**`rabbitmq.node.type`**
:   Node type.

type: keyword


**`rabbitmq.node.uptime`**
:   Node uptime.

type: long



## queue [_queue_9]

queue

**`rabbitmq.queue.name`**
:   The name of the queue with non-ASCII characters escaped as in C.

type: keyword


**`rabbitmq.queue.vhost`**
:   Virtual host name with non-ASCII characters escaped as in C.

type: alias

alias to: rabbitmq.vhost


**`rabbitmq.queue.durable`**
:   Whether or not the queue survives server restarts.

type: boolean


**`rabbitmq.queue.auto_delete`**
:   Whether the queue will be deleted automatically when no longer used.

type: boolean


**`rabbitmq.queue.exclusive`**
:   Whether the queue is exclusive (i.e. has owner_pid).

type: boolean


**`rabbitmq.queue.node`**
:   Node name.

type: alias

alias to: rabbitmq.node.name


**`rabbitmq.queue.state`**
:   The state of the queue. Normally *running*, but may be "{syncing, MsgCount}" if the queue is synchronising. Queues which are located on cluster nodes that are currently down will be shown with a status of *down*.

type: keyword


**`rabbitmq.queue.arguments.max_priority`**
:   Maximum number of priority levels for the queue to support.

type: long


**`rabbitmq.queue.consumers.count`**
:   Number of consumers.

type: long


**`rabbitmq.queue.consumers.utilisation.pct`**
:   Fraction of the time (between 0.0 and 1.0) that the queue is able to immediately deliver messages to consumers. This can be less than 1.0 if consumers are limited by network congestion or prefetch count.

type: scaled_float

format: percent


**`rabbitmq.queue.messages.total.count`**
:   Sum of ready and unacknowledged messages (queue depth).

type: long


**`rabbitmq.queue.messages.total.details.rate`**
:   How much the queue depth has changed per second in the most recent sampling interval.

type: float


**`rabbitmq.queue.messages.ready.count`**
:   Number of messages ready to be delivered to clients.

type: long


**`rabbitmq.queue.messages.ready.details.rate`**
:   How much the count of messages ready has changed per second in the most recent sampling interval.

type: float


**`rabbitmq.queue.messages.unacknowledged.count`**
:   Number of messages delivered to clients but not yet acknowledged.

type: long


**`rabbitmq.queue.messages.unacknowledged.details.rate`**
:   How much the count of unacknowledged messages has changed per second in the most recent sampling interval.

type: float


**`rabbitmq.queue.messages.persistent.count`**
:   Total number of persistent messages in the queue (will always be 0 for transient queues).

type: long


**`rabbitmq.queue.memory.bytes`**
:   Bytes of memory consumed by the Erlang process associated with the queue, including stack, heap and internal structures.

type: long

format: bytes


**`rabbitmq.queue.disk.reads.count`**
:   Total number of times messages have been read from disk by this queue since it started.

type: long


**`rabbitmq.queue.disk.writes.count`**
:   Total number of times messages have been written to disk by this queue since it started.

type: long



## shovel [_shovel]

shovel

**`rabbitmq.shovel.name`**
:   The name of the shovel with non-ASCII characters escaped as in C.

type: keyword


**`rabbitmq.shovel.vhost`**
:   Virtual host name with non-ASCII characters escaped as in C.

type: alias

alias to: rabbitmq.vhost


**`rabbitmq.shovel.node`**
:   Node name.

type: alias

alias to: rabbitmq.node.name


**`rabbitmq.shovel.state`**
:   The state of the shovel. Normally *running*, but could be *starting* or *terminated*.

type: keyword


**`rabbitmq.shovel.type`**
:   The type of the shovel. Either *static* or *dynamic*.

type: keyword


