---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-icinga.html
---

# Icinga fields [exported-fields-icinga]

Icinga Module


## icinga [_icinga]


## debug [_debug_2]

Contains fields for the Icinga debug logs.

**`icinga.debug.facility`**
:   Specifies what component of Icinga logged the message.

type: keyword


**`icinga.debug.severity`**
:   type: alias

alias to: log.level


**`icinga.debug.message`**
:   type: alias

alias to: message



## main [_main]

Contains fields for the Icinga main logs.

**`icinga.main.facility`**
:   Specifies what component of Icinga logged the message.

type: keyword


**`icinga.main.severity`**
:   type: alias

alias to: log.level


**`icinga.main.message`**
:   type: alias

alias to: message



## startup [_startup]

Contains fields for the Icinga startup logs.

**`icinga.startup.facility`**
:   Specifies what component of Icinga logged the message.

type: keyword


**`icinga.startup.severity`**
:   type: alias

alias to: log.level


**`icinga.startup.message`**
:   type: alias

alias to: message


