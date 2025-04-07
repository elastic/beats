---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-activemq.html
---

# ActiveMQ fields [exported-fields-activemq]

Module for parsing ActiveMQ log files.


## activemq [_activemq]

**`activemq.caller`**
:   Name of the caller issuing the logging request (class or resource).

type: keyword


**`activemq.thread`**
:   Thread that generated the logging event.

type: keyword


**`activemq.user`**
:   User that generated the logging event.

type: keyword



## audit [_audit]

Fields from ActiveMQ audit logs.


## log [_log]

Fields from ActiveMQ application logs.

**`activemq.log.stack_trace`**
:   type: keyword


