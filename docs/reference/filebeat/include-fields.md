---
navigation_title: "include_fields"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/include-fields.html
---

# Keep fields from events [include-fields]


The `include_fields` processor specifies which fields to export if a certain condition is fulfilled. The condition is optional. If it’s missing, the specified fields are always exported. The `@timestamp`, `@metadata` and `type` fields are always exported, even if they are not defined in the `include_fields` list.

```yaml
processors:
  - include_fields:
      when:
        condition
      fields: ["field1", "field2", ...]
```

See [Conditions](/reference/filebeat/defining-processors.md#conditions) for a list of supported conditions.

You can specify multiple `include_fields` processors under the `processors` section.

::::{note}
If you define an empty list of fields under `include_fields`, then only the required fields, `@timestamp` and `type`, are exported.
::::


