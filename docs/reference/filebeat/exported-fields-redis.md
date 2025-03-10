---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-redis.html
---

# Redis fields [exported-fields-redis]

Redis Module


## redis [_redis]


## log [_log_13]

Redis log files

**`redis.log.role`**
:   The role of the Redis instance. Can be one of `master`, `slave`, `child` (for RDF/AOF writing child), or `sentinel`.

type: keyword


**`redis.log.pid`**
:   type: alias

alias to: process.pid


**`redis.log.level`**
:   type: alias

alias to: log.level


**`redis.log.message`**
:   type: alias

alias to: message



## slowlog [_slowlog_4]

Slow logs are retrieved from Redis via a network connection.

**`redis.slowlog.cmd`**
:   The command executed.

type: keyword


**`redis.slowlog.duration.us`**
:   How long it took to execute the command in microseconds.

type: long


**`redis.slowlog.id`**
:   The ID of the query.

type: long


**`redis.slowlog.key`**
:   The key on which the command was executed.

type: keyword


**`redis.slowlog.args`**
:   The arguments with which the command was called.

type: keyword


