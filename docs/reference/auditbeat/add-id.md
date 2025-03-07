---
navigation_title: "add_id"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/add-id.html
---

# Generate an ID for an event [add-id]


The `add_id` processor generates a unique ID for an event.

```yaml
processors:
  - add_id: ~
```

The following settings are supported:

`target_field`
:   (Optional) Field where the generated ID will be stored. Default is `@metadata._id`.

`type`
:   (Optional) Type of ID to generate. Currently only `elasticsearch` is supported and is the default. The `elasticsearch` type generates IDs using the same algorithm that Elasticsearch uses for auto-generating document IDs.

