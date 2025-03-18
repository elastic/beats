---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-log.html
---

# Log file content fields [exported-fields-log]

Contains log file lines.

**`log.source.address`**
:   Source address from which the log event was read / sent from.

type: keyword

required: False


**`log.offset`**
:   The file offset the reported line starts at.

type: long

required: False


**`stream`**
:   Log stream when reading container logs, can be *stdout* or *stderr*

type: keyword

required: False


**`input.type`**
:   The input type from which the event was generated. This field is set to the value specified for the `type` option in the input section of the Filebeat config file.

required: True


**`syslog.facility`**
:   The facility extracted from the priority.

type: long

required: False


**`syslog.priority`**
:   The priority of the syslog event.

type: long

required: False


**`syslog.severity_label`**
:   The human readable severity.

type: keyword

required: False


**`syslog.facility_label`**
:   The human readable facility.

type: keyword

required: False


**`process.program`**
:   The name of the program.

type: keyword

required: False


**`log.flags`**
:   This field contains the flags of the event.


**`http.response.content_length`**
:   type: alias

alias to: http.response.body.bytes


**`user_agent.os.full_name`**
:   type: keyword


**`fileset.name`**
:   The Filebeat fileset that generated this event.

type: keyword


**`fileset.module`**
:   type: alias

alias to: event.module


**`read_timestamp`**
:   type: alias

alias to: event.created


**`docker.attrs`**
:   docker.attrs contains labels and environment variables written by docker’s JSON File logging driver. These fields are only available when they are configured in the logging driver options.

type: object


**`icmp.code`**
:   ICMP code.

type: keyword


**`icmp.type`**
:   ICMP type.

type: keyword


**`igmp.type`**
:   IGMP type.

type: keyword


**`azure.eventhub`**
:   Name of the eventhub.

type: keyword


**`azure.offset`**
:   The offset.

type: long


**`azure.enqueued_time`**
:   The enqueued time.

type: date


**`azure.partition_id`**
:   The partition id.

type: long


**`azure.consumer_group`**
:   The consumer group.

type: keyword


**`azure.sequence_number`**
:   The sequence number.

type: long


**`kafka.topic`**
:   Kafka topic

type: keyword


**`kafka.partition`**
:   Kafka partition number

type: long


**`kafka.offset`**
:   Kafka offset of this message

type: long


**`kafka.key`**
:   Kafka key, corresponding to the Kafka value stored in the message

type: keyword


**`kafka.block_timestamp`**
:   Kafka outer (compressed) block timestamp

type: date


**`kafka.headers`**
:   An array of Kafka header strings for this message, in the form "<key>: <value>".

type: array


