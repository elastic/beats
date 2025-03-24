---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/metadata-missing.html
---

# @metadata is missing in Logstash [metadata-missing]

{{ls}} outputs remove `@metadata` fields automatically. Therefore, if {{ls}} instances are chained directly or via some message queue (for example, Redis or Kafka), the `@metadata` field will not be available in the final {{ls}} instance.

::::{tip}
To preserve `@metadata` fields, use the {{ls}} mutate filter with the rename setting to rename the fields to non-internal fields.
::::


