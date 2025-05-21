---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-kafka.html
---

# Kafka fields [exported-fields-kafka]

Kafka module


## kafka [_kafka]


## log [_log_5]

Kafka log lines.

**`kafka.log.component`**
:   Component the log is coming from.

type: keyword


**`kafka.log.class`**
:   Java class the log is coming from.

type: keyword


**`kafka.log.thread`**
:   Thread name the log is coming from.

type: keyword



## trace [_trace]

Trace in the log line.

**`kafka.log.trace.class`**
:   Java class the trace is coming from.

type: keyword


**`kafka.log.trace.message`**
:   Message part of the trace.

type: text


