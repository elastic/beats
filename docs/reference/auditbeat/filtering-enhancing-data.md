---
navigation_title: "Processors"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/filtering-and-enhancing-data.html
---

# Filter and enhance data with processors [filtering-and-enhancing-data]


You can [define processors](/reference/auditbeat/defining-processors.md) in your configuration to process events before they are sent to the configured output. The libbeat library provides processors for:

* reducing the number of exported fields
* enhancing events with additional metadata
* performing additional processing and decoding

Each processor receives an event, applies a defined action to the event, and returns the event. If you define a list of processors, they are executed in the order they are defined in the Auditbeat configuration file.

```yaml
event -> processor 1 -> event1 -> processor 2 -> event2 ...
```

::::{important}
It’s recommended to do all drop and renaming of existing fields as the last step in a processor configuration. This is because dropping or renaming fields can remove data necessary for the next processor in the chain, for example dropping the `source.ip` field would remove one of the fields necessary for the `community_id` processor to function. If it’s necessary to remove, rename or overwrite an existing event field, please make sure it’s done by a corresponding processor ([`drop_fields`](/reference/auditbeat/drop-fields.md), [`rename`](/reference/auditbeat/rename-fields.md) or [`add_fields`](/reference/auditbeat/add-fields.md)) placed at the end of the processor list defined in the input configuration.
::::














































