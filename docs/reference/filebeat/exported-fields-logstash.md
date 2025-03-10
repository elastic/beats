---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-logstash.html
---

# logstash fields [exported-fields-logstash]

logstash Module


## logstash [_logstash]


## log [_log_7]

Fields from the Logstash logs.

**`logstash.log.module`**
:   The module or class where the event originate.

type: keyword


**`logstash.log.thread`**
:   Information about the running thread where the log originate.

type: keyword


**`logstash.log.thread.text`**
:   type: text


**`logstash.log.log_event`**
:   key and value debugging information.

type: object


**`logstash.log.log_event.action`**
:   type: keyword


**`logstash.log.pipeline_id`**
:   The ID of the pipeline.

type: keyword

example: main


**`logstash.log.message`**
:   type: alias

alias to: message


**`logstash.log.level`**
:   type: alias

alias to: log.level



## slowlog [_slowlog_2]

slowlog

**`logstash.slowlog.module`**
:   The module or class where the event originate.

type: keyword


**`logstash.slowlog.thread`**
:   Information about the running thread where the log originate.

type: keyword


**`logstash.slowlog.thread.text`**
:   type: text


**`logstash.slowlog.event`**
:   Raw dump of the original event

type: keyword


**`logstash.slowlog.event.text`**
:   type: text


**`logstash.slowlog.plugin_name`**
:   Name of the plugin

type: keyword


**`logstash.slowlog.plugin_type`**
:   Type of the plugin: Inputs, Filters, Outputs or Codecs.

type: keyword


**`logstash.slowlog.took_in_millis`**
:   Execution time for the plugin in milliseconds.

type: long


**`logstash.slowlog.plugin_params`**
:   String value of the plugin configuration

type: keyword


**`logstash.slowlog.plugin_params.text`**
:   type: text


**`logstash.slowlog.plugin_params_object`**
:   key → value of the configuration used by the plugin.

type: object


**`logstash.slowlog.level`**
:   type: alias

alias to: log.level


**`logstash.slowlog.took_in_nanos`**
:   type: alias

alias to: event.duration


