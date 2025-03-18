---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-beat-common.html
---

# Beat fields [exported-fields-beat-common]

Contains common beat fields available in all event types.

**`agent.hostname`**
:   Deprecated - use agent.name or agent.id to identify an agent.

type: alias

alias to: agent.name


**`beat.timezone`**
:   type: alias

alias to: event.timezone


**`fields`**
:   Contains user configurable fields.

type: object


**`beat.name`**
:   type: alias

alias to: host.name


**`beat.hostname`**
:   type: alias

alias to: agent.name


**`timeseries.instance`**
:   Time series instance id

type: keyword


