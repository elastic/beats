---
navigation_title: "add_fields"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/add-fields.html
---

# Add fields [add-fields]


The `add_fields` processor adds additional fields to the event.  Fields can be scalar values, arrays, dictionaries, or any nested combination of these. The `add_fields` processor will overwrite the target field if it already exists. By default the fields that you specify will be grouped under the `fields` sub-dictionary in the event. To group the fields under a different sub-dictionary, use the `target` setting. To store the fields as top-level fields, set `target: ''`.

`target`
:   (Optional) Sub-dictionary to put all fields into. Defaults to `fields`. Setting this to `@metadata` will add values to the event metadata instead of fields.

`fields`
:   Fields to be added.

For example, this configuration:

```yaml
processors:
  - add_fields:
      target: project
      fields:
        name: myproject
        id: '574734885120952459'
```

Adds these fields to any event:

```json
{
  "project": {
    "name": "myproject",
    "id": "574734885120952459"
  }
}
```

This configuration will alter the event metadata:

```yaml
processors:
  - add_fields:
      target: '@metadata'
      fields:
        op_type: "index"
```

When the event is ingested (e.g. by Elastisearch) the document will have `op_type: "index"` set as a metadata field.

