---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-pensando.html
applies_to:
  stack: ga
  serverless: ga
---

% This file is generated! See dev-tools/mage/generate_fields_docs.go

# Pensando fields [exported-fields-pensando]

pensando Module

## pensando [_pensando]

Fields from Pensando logs.

## dfw [_dfw]

```{applies_to}
stack: beta
```

Fields for Pensando DFW

**`pensando.dfw.action`**
:   Action on the flow.

    type: keyword


**`pensando.dfw.app_id`**
:   Application ID

    type: integer


**`pensando.dfw.destination_address`**
:   Address of destination.

    type: keyword


**`pensando.dfw.destination_port`**
:   Port of destination.

    type: integer


**`pensando.dfw.direction`**
:   Direction of the flow

    type: keyword


**`pensando.dfw.protocol`**
:   Protocol of the flow

    type: keyword


**`pensando.dfw.rule_id`**
:   Rule ID that was matched.

    type: keyword


**`pensando.dfw.session_id`**
:   Session ID of the flow

    type: integer


**`pensando.dfw.session_state`**
:   Session state of the flow.

    type: keyword


**`pensando.dfw.source_address`**
:   Source address of the flow.

    type: keyword


**`pensando.dfw.source_port`**
:   Source port of the flow.

    type: integer


**`pensando.dfw.timestamp`**
:   Timestamp of the log.

    type: date


