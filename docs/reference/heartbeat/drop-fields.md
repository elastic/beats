---
navigation_title: "drop_fields"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/drop-fields.html
---

# Drop fields from events [drop-fields]


The `drop_fields` processor specifies which fields to drop if a certain condition is fulfilled. The condition is optional. If it’s missing, the specified fields are always dropped. The `@timestamp` and `type` fields cannot be dropped, even if they show up in the `drop_fields` list.

```yaml
processors:
  - drop_fields:
      when:
        condition
      fields: ["field1", "field2", ...]
      ignore_missing: false
```

See [Conditions](/reference/heartbeat/defining-processors.md#conditions) for a list of supported conditions.

::::{note}
If you define an empty list of fields under `drop_fields`, then no fields are dropped.
::::


The `drop_fields` processor has the following configuration settings:

`fields`
:   If non-empty, a list of matching field names will be removed. Any element in array can contain a regular expression delimited by two slashes (*/reg_exp/*), in order to match (name) and remove more than one field.

`ignore_missing`
:   (Optional) If `true` the processor will not return an error when a specified field does not exist. Defaults to `false`.

