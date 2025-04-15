---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/exported-fields-state.html
---

# Monitor state fields [exported-fields-state]

state related fields


## state [_state]

Present in the last event emitted during a check. If a monitor checks multiple endpoints, as is the case with `mode: all`.

**`state.id`**
:   ID of this state

type: keyword


**`state.started_at`**
:   First time state with this ID was seen

type: date


**`state.duration_ms`**
:   Length of time this state has existed in millis

type: long


**`state.status`**
:   The current status, "up", "down", or "flapping" any state can change into flapping.

type: keyword


**`state.checks`**
:   total checks run

type: integer


**`state.up`**
:   total up checks run

type: integer


**`state.down`**
:   total down checks run

type: integer


**`state.flap_history`**
:   Object is not enabled.


**`state.ends`**
:   the state that was ended by this state

type: object


