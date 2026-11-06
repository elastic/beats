---
description: "Use the drop_event processor in Metricbeat to drop entire events that match a condition, so they are never sent to the output."
navigation_title: "drop_event"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/drop-event.html
applies_to:
  stack: ga
  serverless: ga
---

# Drop events in Metricbeat [drop-event]


The `drop_event` processor drops the entire event if the associated condition is fulfilled. The condition is mandatory, because without one, all the events are dropped.

```yaml
processors:
  - drop_event:
      when:
        condition
```

See [Conditions](/reference/metricbeat/defining-processors.md#conditions) for a list of supported conditions.

