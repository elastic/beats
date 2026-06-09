---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-kafka.html
applies_to:
  stack: ga
  serverless: ga
---

% This file is generated! See dev-tools/mage/generate_fields_docs.go

# Kafka fields [exported-fields-kafka]

Kafka module

## kafka [_kafka]



## log [_log]

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


