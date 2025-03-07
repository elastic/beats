---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/exported-fields-amqp.html
---

# AMQP fields [exported-fields-amqp]

AMQP specific event fields.

**`amqp.reply-code`**
:   AMQP reply code to an error, similar to http reply-code

type: long

example: 404


**`amqp.reply-text`**
:   Text explaining the error.

type: keyword


**`amqp.class-id`**
:   Failing method class.

type: long


**`amqp.method-id`**
:   Failing method ID.

type: long


**`amqp.exchange`**
:   Name of the exchange.

type: keyword


**`amqp.exchange-type`**
:   Exchange type.

type: keyword

example: fanout


**`amqp.passive`**
:   If set, do not create exchange/queue.

type: boolean


**`amqp.durable`**
:   If set, request a durable exchange/queue.

type: boolean


**`amqp.exclusive`**
:   If set, request an exclusive queue.

type: boolean


**`amqp.auto-delete`**
:   If set, auto-delete queue when unused.

type: boolean


**`amqp.no-wait`**
:   If set, the server will not respond to the method.

type: boolean


**`amqp.consumer-tag`**
:   Identifier for the consumer, valid within the current channel.


**`amqp.delivery-tag`**
:   The server-assigned and channel-specific delivery tag.

type: long


**`amqp.message-count`**
:   The number of messages in the queue, which will be zero for newly-declared queues.

type: long


**`amqp.consumer-count`**
:   The number of consumers of a queue.

type: long


**`amqp.routing-key`**
:   Message routing key.

type: keyword


**`amqp.no-ack`**
:   If set, the server does not expect acknowledgements for messages.

type: boolean


**`amqp.no-local`**
:   If set, the server will not send messages to the connection that published them.

type: boolean


**`amqp.if-unused`**
:   Delete only if unused.

type: boolean


**`amqp.if-empty`**
:   Delete only if empty.

type: boolean


**`amqp.queue`**
:   The queue name identifies the queue within the vhost.

type: keyword


**`amqp.redelivered`**
:   Indicates that the message has been previously delivered to this or another client.

type: boolean


**`amqp.multiple`**
:   Acknowledge multiple messages.

type: boolean


**`amqp.arguments`**
:   Optional additional arguments passed to some methods. Can be of various types.

type: object


**`amqp.mandatory`**
:   Indicates mandatory routing.

type: boolean


**`amqp.immediate`**
:   Request immediate delivery.

type: boolean


**`amqp.content-type`**
:   MIME content type.

type: keyword

example: text/plain


**`amqp.content-encoding`**
:   MIME content encoding.

type: keyword


**`amqp.headers`**
:   Message header field table.

type: object


**`amqp.delivery-mode`**
:   Non-persistent (1) or persistent (2).

type: keyword


**`amqp.priority`**
:   Message priority, 0 to 9.

type: long


**`amqp.correlation-id`**
:   Application correlation identifier.

type: keyword


**`amqp.reply-to`**
:   Address to reply to.

type: keyword


**`amqp.expiration`**
:   Message expiration specification.

type: keyword


**`amqp.message-id`**
:   Application message identifier.

type: keyword


**`amqp.timestamp`**
:   Message timestamp.

type: keyword


**`amqp.type`**
:   Message type name.

type: keyword


**`amqp.user-id`**
:   Creating user id.

type: keyword


**`amqp.app-id`**
:   Creating application id.

type: keyword


