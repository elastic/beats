---
navigation_title: "Processors"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/filtering-and-enhancing-data.html
---

# Filter and enhance data with processors [filtering-and-enhancing-data]


You can [define processors](/reference/winlogbeat/defining-processors.md) in your configuration to process events before they are sent to the configured output. The libbeat library provides processors for:

* reducing the number of exported fields
* enhancing events with additional metadata
* performing additional processing and decoding

Each processor receives an event, applies a defined action to the event, and returns the event. If you define a list of processors, they are executed in the order they are defined in the Winlogbeat configuration file.

```yaml
event -> processor 1 -> event1 -> processor 2 -> event2 ...
```

::::{important}
It’s recommended to do all drop and renaming of existing fields as the last step in a processor configuration. This is because dropping or renaming fields can remove data necessary for the next processor in the chain, for example dropping the `source.ip` field would remove one of the fields necessary for the `community_id` processor to function. If it’s necessary to remove, rename or overwrite an existing event field, please make sure it’s done by a corresponding processor ([`drop_fields`](/reference/winlogbeat/drop-fields.md), [`rename`](/reference/winlogbeat/rename-fields.md) or [`add_fields`](/reference/winlogbeat/add-fields.md)) placed at the end of the processor list defined in the input configuration.
::::


For example, the following filter configuration drops a few fields that are rarely used (`provider_guid`, `process_id`, `thread_id`, and `version`) and one nested field, `event_data.ErrorSourceTable`:

```yaml
processors:
  - drop_fields:
      fields: [winlog.provider_guid, winlog.process.pid, winlog.process.thread.id, winlog.version, winlog.event_data.ErrorSourceTable]
```














































