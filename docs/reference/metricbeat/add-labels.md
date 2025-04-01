---
navigation_title: "add_labels"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/add-labels.html
---

# Add labels [add-labels]


The `add_labels` processors adds a set of key-value pairs to an event. The processor will flatten nested configuration objects like arrays or dictionaries into a fully qualified name by merging nested names with a `.`. Array entries create numeric names starting with 0.  Labels are always stored under the Elastic Common Schema compliant `labels` sub-dictionary.

`labels`
:   dictionaries of labels to be added.

For example, this configuration:

```yaml
processors:
  - add_labels:
      labels:
        number: 1
        with.dots: test
        nested:
          with.dots: nested
        array:
          - do
          - re
          - with.field: mi
```

Adds these fields to every event:

```json
{
  "labels": {
    "number": 1,
    "with.dots": "test",
    "nested.with.dots": "nested",
    "array.0": "do",
    "array.1": "re",
    "array.2.with.field": "mi"
  }
}
```

