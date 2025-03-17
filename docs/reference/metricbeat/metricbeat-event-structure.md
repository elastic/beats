---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-event-structure.html
---

# Event structure [metricbeat-event-structure]

Every event sent by Metricbeat has the same basic structure. It contains the following fields:

**`@timestamp`**
:   Time when the event was captured

**`host.hostname`**
:   Hostname of the server on which the Beat is running

**`agent.type`**
:   Name given to the Beat

**`event.module`**
:   Name of the module that the data is from

**`event.dataset`**
:   Name of the module that the data is from

For example:

```json
{
  "@timestamp": "2016-06-22T22:05:53.291Z",
  "agent": {
    "type": "metricbeat"
  },
  "host": {
     "hostname": "host.example.com",
   },
  "event": {
    "dataset": "system.process",
    "module": process
  },
  .
  .
  .

  "type": "metricsets"
}
```

For more information about the exported fields, see [Exported fields](/reference/metricbeat/exported-fields.md).

