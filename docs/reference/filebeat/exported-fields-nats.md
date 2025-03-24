---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-nats.html
---

# NATS fields [exported-fields-nats]

Module for parsing NATS log files.


## nats [_nats]

Fields from NATS logs.


## log [_log_10]

Nats log files


## client [_client_3]

Fields from NATS logs client.

**`nats.log.client.id`**
:   The id of the client

type: integer



## msg [_msg]

Fields from NATS logs message.

**`nats.log.msg.bytes`**
:   Size of the payload in bytes

type: long

format: bytes


**`nats.log.msg.type`**
:   The protocol message type

type: keyword


**`nats.log.msg.subject`**
:   Subject name this message was received on

type: keyword


**`nats.log.msg.sid`**
:   The unique alphanumeric subscription ID of the subject

type: integer


**`nats.log.msg.reply_to`**
:   The inbox subject on which the publisher is listening for responses

type: keyword


**`nats.log.msg.max_messages`**
:   An optional number of messages to wait for before automatically unsubscribing

type: integer


**`nats.log.msg.error.message`**
:   Details about the error occurred

type: text


**`nats.log.msg.queue_group`**
:   The queue group which subscriber will join

type: text


