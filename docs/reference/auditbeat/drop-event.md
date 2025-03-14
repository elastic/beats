---
navigation_title: "drop_event"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/drop-event.html
---

# Drop events [drop-event]


The `drop_event` processor drops the entire event if the associated condition is fulfilled. The condition is mandatory, because without one, all the events are dropped.

```yaml
processors:
  - drop_event:
      when:
        condition
```

See [Conditions](/reference/auditbeat/defining-processors.md#conditions) for a list of supported conditions.

