---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-ibmmq.html
---

# ibmmq fields [exported-fields-ibmmq]

ibmmq Module


## ibmmq [_ibmmq]


## errorlog [_errorlog]

IBM MQ error logs

**`ibmmq.errorlog.installation`**
:   This is the installation name which can be given at installation time. Each installation of IBM MQ on UNIX, Linux, and Windows, has a unique identifier known as an installation name. The installation name is used to associate things such as queue managers and configuration files with an installation.

type: keyword


**`ibmmq.errorlog.qmgr`**
:   Name of the queue manager. Queue managers provide queuing services to applications, and manages the queues that belong to them.

type: keyword


**`ibmmq.errorlog.arithinsert`**
:   Changing content based on error.id

type: keyword


**`ibmmq.errorlog.commentinsert`**
:   Changing content based on error.id

type: keyword


**`ibmmq.errorlog.errordescription`**
:   Please add description

type: text

example: Please add example


**`ibmmq.errorlog.explanation`**
:   Explaines the error in more detail

type: keyword


**`ibmmq.errorlog.action`**
:   Defines what to do when the error occurs

type: keyword


**`ibmmq.errorlog.code`**
:   Error code.

type: keyword


