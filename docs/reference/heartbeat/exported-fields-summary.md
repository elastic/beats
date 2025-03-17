---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/exported-fields-summary.html
---

# Monitor summary fields [exported-fields-summary]

None


## summary [_summary]

Present in the last event emitted during a check. If a monitor checks multiple endpoints, as is the case with `mode: all`.

**`summary.up`**
:   The number of endpoints that succeeded

type: integer


**`summary.down`**
:   The number of endpoints that failed

type: integer


**`summary.status`**
:   The status of this check as a whole. Either up or down.

type: keyword


**`summary.attempt`**
:   When performing a check this number is 1 for the first check, and increments in the event of a retry.

type: short


**`summary.max_attempts`**
:   The maximum number of checks that may be performed. Note, the actual number may be smaller.

type: short


**`summary.final_attempt`**
:   True if no further checks will be performed in this retry group.

type: boolean


**`summary.retry_group`**
:   A unique token used to group checks across attempts.

type: keyword


