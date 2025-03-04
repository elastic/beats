---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-kafka.html
---

# Kafka fields [exported-fields-kafka]

Kafka module


## kafka [_kafka]


## broker [_broker_2]

Broker Consumer Group Information have been read from (Broker handling the consumer group).

**`kafka.broker.id`**
:   Broker id

type: long


**`kafka.broker.address`**
:   Broker advertised address

type: keyword


**`kafka.topic.name`**
:   Topic name

type: keyword


**`kafka.topic.error.code`**
:   Topic error code.

type: long


**`kafka.partition.id`**
:   Partition id.

type: long


**`kafka.partition.topic_id`**
:   Unique id of the partition in the topic.

type: keyword


**`kafka.partition.topic_broker_id`**
:   Unique id of the partition in the topic and the broker.

type: keyword



## broker [_broker_3]

Broker metrics from Kafka Broker JMX

**`kafka.broker.mbean`**
:   Mbean that this event is related to

type: keyword


**`kafka.broker.request.channel.queue.size`**
:   The size of the request queue

type: long


**`kafka.broker.request.produce.failed_per_second`**
:   The rate of failed produce requests per second

type: float


**`kafka.broker.request.fetch.failed_per_second`**
:   The rate of client fetch request failures per second

type: float


**`kafka.broker.request.produce.failed`**
:   The number of failed produce requests

type: float


**`kafka.broker.request.fetch.failed`**
:   The number of client fetch request failures

type: float


**`kafka.broker.replication.leader_elections`**
:   The leader election rate

type: float


**`kafka.broker.replication.unclean_leader_elections`**
:   The unclean leader election rate

type: float


**`kafka.broker.session.zookeeper.disconnect`**
:   The ZooKeeper closed sessions per second

type: float


**`kafka.broker.session.zookeeper.expire`**
:   The ZooKeeper expired sessions per second

type: float


**`kafka.broker.session.zookeeper.readonly`**
:   The ZooKeeper readonly sessions per second

type: float


**`kafka.broker.session.zookeeper.sync`**
:   The ZooKeeper client connections per second

type: float


**`kafka.broker.log.flush_rate`**
:   The log flush rate

type: float


**`kafka.broker.topic.net.in.bytes_per_sec`**
:   The incoming byte rate per topic

type: float


**`kafka.broker.topic.net.out.bytes_per_sec`**
:   The outgoing byte rate per topic

type: float


**`kafka.broker.topic.net.rejected.bytes_per_sec`**
:   The rejected byte rate per topic

type: float


**`kafka.broker.topic.messages_in`**
:   The incoming message rate per topic

type: float


**`kafka.broker.net.in.bytes_per_sec`**
:   The incoming byte rate

type: float


**`kafka.broker.net.out.bytes_per_sec`**
:   The outgoing byte rate

type: float


**`kafka.broker.net.rejected.bytes_per_sec`**
:   The rejected byte rate

type: float


**`kafka.broker.messages_in`**
:   The incoming message rate

type: float



## consumer [_consumer]

Consumer metrics from Kafka Consumer JMX

**`kafka.consumer.mbean`**
:   Mbean that this event is related to

type: keyword


**`kafka.consumer.fetch_rate`**
:   The minimum rate at which the consumer sends fetch requests to a broker

type: float


**`kafka.consumer.bytes_consumed`**
:   The average number of bytes consumed for a specific topic per second

type: float


**`kafka.consumer.records_consumed`**
:   The average number of records consumed per second for a specific topic

type: float


**`kafka.consumer.in.bytes_per_sec`**
:   The rate of bytes coming in to the consumer

type: float


**`kafka.consumer.max_lag`**
:   The maximum consumer lag

type: float


**`kafka.consumer.zookeeper_commits`**
:   The rate of offset commits to ZooKeeper

type: float


**`kafka.consumer.kafka_commits`**
:   The rate of offset commits to Kafka

type: float


**`kafka.consumer.messages_in`**
:   The rate of consumer message consumption

type: float



## consumergroup [_consumergroup]

consumergroup

**`kafka.consumergroup.id`**
:   Consumer Group ID

type: keyword


**`kafka.consumergroup.offset`**
:   consumer offset into partition being read

type: long


**`kafka.consumergroup.meta`**
:   custom consumer meta data string

type: keyword


**`kafka.consumergroup.consumer_lag`**
:   consumer lag for partition/topic calculated as the difference between the partition offset and consumer offset

type: long


**`kafka.consumergroup.error.code`**
:   kafka consumer/partition error code.

type: long



## client [_client_3]

Assigned client reading events from partition

**`kafka.consumergroup.client.id`**
:   Client ID (kafka setting client.id)

type: keyword


**`kafka.consumergroup.client.host`**
:   Client host

type: keyword


**`kafka.consumergroup.client.member_id`**
:   internal consumer group member ID

type: keyword



## partition [_partition_2]

partition


## offset [_offset]

Available offsets of the given partition.

**`kafka.partition.offset.newest`**
:   Newest offset of the partition.

type: long


**`kafka.partition.offset.oldest`**
:   Oldest offset of the partition.

type: long



## partition [_partition_3]

Partition data.

**`kafka.partition.partition.leader`**
:   Leader id (broker).

type: long


**`kafka.partition.partition.replica`**
:   Replica id (broker).

type: long


**`kafka.partition.partition.insync_replica`**
:   Indicates if replica is included in the in-sync replicate set (ISR).

type: boolean


**`kafka.partition.partition.is_leader`**
:   Indicates if replica is the leader

type: boolean


**`kafka.partition.partition.error.code`**
:   Error code from fetching partition.

type: long



## producer [_producer]

Producer metrics from Kafka Producer JMX

**`kafka.producer.mbean`**
:   Mbean that this event is related to

type: keyword


**`kafka.producer.available_buffer_bytes`**
:   The total amount of buffer memory

type: float


**`kafka.producer.batch_size_avg`**
:   The average number of bytes sent

type: float


**`kafka.producer.batch_size_max`**
:   The maximum number of bytes sent

type: long


**`kafka.producer.record_send_rate`**
:   The average number of records sent per second

type: float


**`kafka.producer.record_retry_rate`**
:   The average number of retried record sends per second

type: float


**`kafka.producer.record_error_rate`**
:   The average number of retried record sends per second

type: float


**`kafka.producer.records_per_request`**
:   The average number of records sent per second

type: float


**`kafka.producer.record_size_avg`**
:   The average record size

type: float


**`kafka.producer.record_size_max`**
:   The maximum record size

type: long


**`kafka.producer.request_rate`**
:   The number of producer requests per second

type: float


**`kafka.producer.response_rate`**
:   The number of producer responses per second

type: float


**`kafka.producer.io_wait`**
:   The producer I/O wait time

type: float


**`kafka.producer.out.bytes_per_sec`**
:   The rate of bytes going out for the producer

type: float


**`kafka.producer.message_rate`**
:   The producer message rate

type: float


