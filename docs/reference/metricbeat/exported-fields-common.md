---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/exported-fields-common.html
---

# Common fields [exported-fields-common]

Contains common fields available in all event types.

**`metricset.module`**
:   The name of the module that generated the event.

type: alias

alias to: event.module


**`metricset.name`**
:   The name of the metricset that generated the event.


**`metricset.period`**
:   Current data collection period for this event in milliseconds.

type: integer


**`service.hostname`**
:   Host name of the machine where the service is running.


**`type`**
:   The document type. Always set to "doc".

example: metricsets

required: True


**`systemd.fragment_path`**
:   the location of the systemd unit path

type: keyword


**`systemd.unit`**
:   the unit name of the systemd service

type: keyword


