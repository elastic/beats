---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-activemq.html
---

# ActiveMQ fields [exported-fields-activemq]

activemq module


## activemq [_activemq]


## broker [_broker]

Broker metrics from org.apache.activemq:brokerName=*,type=Broker

**`activemq.broker.mbean`**
:   Mbean that this event is related to

type: keyword


**`activemq.broker.name`**
:   Broker name

type: keyword


**`activemq.broker.memory.broker.pct`**
:   The percentage of the memory limit used.

type: scaled_float

format: percent


**`activemq.broker.memory.store.pct`**
:   Percent of store limit used.

type: scaled_float

format: percent


**`activemq.broker.memory.temp.pct`**
:   The percentage of the temp usage limit used.

type: scaled_float

format: percent


**`activemq.broker.connections.count`**
:   Total number of connections.

type: long


**`activemq.broker.consumers.count`**
:   Number of message consumers.

type: long


**`activemq.broker.messages.dequeue.count`**
:   Number of messages that have been acknowledged on the broker.

type: long


**`activemq.broker.messages.enqueue.count`**
:   Number of messages that have been sent to the destination.

type: long


**`activemq.broker.messages.count`**
:   Number of unacknowledged messages on the broker.

type: long


**`activemq.broker.producers.count`**
:   Number of message producers active on destinations on the broker.

type: long



## queue [_queue_7]

Queue metrics from org.apache.activemq:brokerName=**,destinationName=**,destinationType=Queue,type=Broker

**`activemq.queue.mbean`**
:   Mbean that this event is related to

type: keyword


**`activemq.queue.name`**
:   Queue name

type: keyword


**`activemq.queue.size`**
:   Queue size

type: long


**`activemq.queue.messages.enqueue.time.avg`**
:   Average time a message was held on this destination.

type: double


**`activemq.queue.messages.size.avg`**
:   Average message size on this destination.

type: long


**`activemq.queue.consumers.count`**
:   Number of consumers subscribed to this destination.

type: long


**`activemq.queue.messages.dequeue.count`**
:   Number of messages that has been acknowledged (and removed) from the destination.

type: long


**`activemq.queue.messages.dispatch.count`**
:   Number of messages that has been delivered to consumers, including those not acknowledged.

type: long


**`activemq.queue.messages.enqueue.count`**
:   Number of messages that have been sent to the destination.

type: long


**`activemq.queue.messages.expired.count`**
:   Number of messages that have been expired.

type: long


**`activemq.queue.messages.inflight.count`**
:   Number of messages that have been dispatched to, but not acknowledged by, consumers.

type: long


**`activemq.queue.messages.enqueue.time.max`**
:   The longest time a message was held on this destination.

type: long


**`activemq.queue.memory.broker.pct`**
:   Percent of memory limit used.

type: scaled_float

format: percent


**`activemq.queue.messages.enqueue.time.min`**
:   The shortest time a message was held on this destination.

type: long


**`activemq.queue.producers.count`**
:   Number of producers attached to this destination.

type: long



## topic [_topic]

Topic metrics from org.apache.activemq:brokerName=**,destinationName=**,destinationType=Topic,type=Broker

**`activemq.topic.mbean`**
:   Mbean that this event is related to

type: keyword


**`activemq.topic.name`**
:   Topic name

type: keyword


**`activemq.topic.messages.enqueue.time.avg`**
:   Average time a message was held on this destination.

type: double


**`activemq.topic.messages.size.avg`**
:   Average message size on this destination.

type: long


**`activemq.topic.consumers.count`**
:   Number of consumers subscribed to this destination.

type: long


**`activemq.topic.messages.dequeue.count`**
:   Number of messages that has been acknowledged (and removed) from the destination.

type: long


**`activemq.topic.messages.dispatch.count`**
:   Number of messages that has been delivered to consumers, including those not acknowledged.

type: long


**`activemq.topic.messages.enqueue.count`**
:   Number of messages that have been sent to the destination.

type: long


**`activemq.topic.messages.expired.count`**
:   Number of messages that have been expired.

type: long


**`activemq.topic.messages.inflight.count`**
:   Number of messages that have been dispatched to, but not acknowledged by, consumers.

type: long


**`activemq.topic.messages.enqueue.time.max`**
:   The longest time a message was held on this destination.

type: long


**`activemq.topic.memory.broker.pct`**
:   Percent of memory limit used.

type: scaled_float

format: percent


**`activemq.topic.messages.enqueue.time.min`**
:   The shortest time a message was held on this destination.

type: long


**`activemq.topic.producers.count`**
:   Number of producers attached to this destination.

type: long


