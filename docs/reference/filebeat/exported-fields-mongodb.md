---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-mongodb.html
---

# mongodb fields [exported-fields-mongodb]

Module for parsing MongoDB log files.


## mongodb [_mongodb]

Fields from MongoDB logs.


## log [_log_8]

Contains fields from MongoDB logs.

**`mongodb.log.component`**
:   Functional categorization of message

type: keyword

example: COMMAND


**`mongodb.log.context`**
:   Context of message

type: keyword

example: initandlisten


**`mongodb.log.severity`**
:   type: alias

alias to: log.level


**`mongodb.log.message`**
:   type: alias

alias to: message


**`mongodb.log.id`**
:   Integer representing the unique identifier of the log statement

type: long

example: 4615611


